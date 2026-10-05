#!/usr/bin/env python3
"""Verify every OCI descriptor/blob and the exact Linux platform set."""
import hashlib
import json
from pathlib import Path, PurePosixPath
import re
import sys
import tarfile

INDEX_TYPES = {'application/vnd.oci.image.index.v1+json', 'application/vnd.docker.distribution.manifest.list.v2+json'}
MANIFEST_TYPES = {'application/vnd.oci.image.manifest.v1+json', 'application/vnd.docker.distribution.manifest.v2+json'}
CONFIG_TYPES = {'application/vnd.oci.image.config.v1+json', 'application/vnd.docker.container.image.v1+json'}
LAYER_TYPES = {
    'application/vnd.oci.image.layer.v1.tar', 'application/vnd.oci.image.layer.v1.tar+gzip', 'application/vnd.oci.image.layer.v1.tar+zstd',
    'application/vnd.oci.image.layer.nondistributable.v1.tar', 'application/vnd.oci.image.layer.nondistributable.v1.tar+gzip', 'application/vnd.oci.image.layer.nondistributable.v1.tar+zstd',
    'application/vnd.docker.image.rootfs.diff.tar', 'application/vnd.docker.image.rootfs.diff.tar.gzip', 'application/vnd.docker.image.rootfs.foreign.diff.tar.gzip',
}
PLATFORMS = {('linux', 'amd64'), ('linux', 'arm64')}
MAX_TOTAL = 4 * 1024 * 1024 * 1024
MAX_JSON = 16 * 1024 * 1024


def require(condition, message):
    if not condition:
        raise ValueError(message)


def unique_object(pairs):
    result = {}
    for key, value in pairs:
        require(key not in result, 'duplicate OCI JSON property: ' + key)
        result[key] = value
    return result


def read_json(raw):
    require(len(raw) <= MAX_JSON, 'OCI JSON exceeds limit')
    value = json.loads(raw, object_pairs_hook=unique_object)
    require(isinstance(value, dict), 'OCI JSON must be an object')
    return value


def sha256(path):
    value = hashlib.sha256()
    with Path(path).open('rb') as stream:
        for block in iter(lambda: stream.read(1024 * 1024), b''):
            value.update(block)
    return value.hexdigest()


def inspect(archive, expected_platforms=None):
    """Return verified root bytes/report, without extracting filesystem paths."""
    expected_platforms = PLATFORMS if expected_platforms is None else set(expected_platforms)
    require(expected_platforms and expected_platforms <= PLATFORMS, 'unsupported expected OCI platform set')
    with tarfile.open(archive) as stream:
        members, blobs, total = {}, {}, 0
        for member in stream:
            name = member.name.rstrip('/') if member.isdir() else member.name
            path = PurePosixPath(name)
            require(name and not path.is_absolute() and str(path) == name and '\\' not in name and not any(p in ('.', '..') for p in path.parts), 'unsafe OCI archive path')
            require(name not in members and (member.isfile() or member.isdir()), 'duplicate or unsupported OCI archive member')
            members[name] = member
            require(member.size >= 0, 'negative OCI member size')
            total += member.size
            require(total <= MAX_TOTAL, 'OCI archive expansion exceeds limit')
            if member.isdir():
                require(name in ('blobs', 'blobs/sha256'), 'unexpected OCI directory')
                continue
            require(name in ('index.json', 'oci-layout', 'manifest.json') or re.fullmatch(r'blobs/sha256/[0-9a-f]{64}', name), 'unexpected OCI archive file: ' + name)
            if name.startswith('blobs/'):
                value = hashlib.sha256()
                with stream.extractfile(member) as source:
                    for block in iter(lambda: source.read(1024 * 1024), b''):
                        value.update(block)
                require(value.hexdigest() == name.rsplit('/', 1)[1], 'OCI blob hash mismatch: ' + name)
                blobs['sha256:' + value.hexdigest()] = member
        require('index.json' in members and 'oci-layout' in members, 'missing OCI index/layout')

        def bytes_member(member):
            require(member.isfile() and member.size <= MAX_JSON, 'invalid OCI JSON member')
            with stream.extractfile(member) as source:
                return source.read()

        def json_member(name):
            return read_json(bytes_member(members[name]))

        require(json_member('oci-layout').get('imageLayoutVersion') == '1.0.0', 'unsupported OCI layout version')
        index_raw = bytes_member(members['index.json'])
        index = read_json(index_raw)
        require(index.get('schemaVersion') == 2 and isinstance(index.get('manifests'), list) and index['manifests'], 'invalid OCI archive index')

        def descriptor(item, types):
            require(isinstance(item, dict) and item.get('mediaType') in types, 'unsupported OCI descriptor media type')
            value = item.get('digest')
            require(isinstance(value, str) and re.fullmatch(r'sha256:[0-9a-f]{64}', value), 'invalid OCI descriptor digest')
            size = item.get('size')
            require(type(size) is int and size >= 0 and value in blobs and blobs[value].size == size, 'missing OCI blob or descriptor size mismatch')
            return blobs[value]

        reached = set()
        roots = index['manifests']
        require(all(isinstance(r, dict) for r in roots), 'invalid OCI root descriptor')
        if len(roots) == 1 and roots[0].get('mediaType') in INDEX_TYPES:
            root = roots[0]
            raw = bytes_member(descriptor(root, INDEX_TYPES))
            doc = read_json(raw)
            require(doc.get('schemaVersion') == 2 and doc.get('mediaType', root['mediaType']) == root['mediaType'], 'invalid OCI root index')
            manifests = doc.get('manifests')
            reached.add(root['digest'])
            root_digest, root_type = root['digest'], root['mediaType']
        elif len(roots) == 1 and len(expected_platforms) == 1:
            root = roots[0]
            raw = bytes_member(descriptor(root, MANIFEST_TYPES))
            manifests = roots
            root_digest, root_type = root['digest'], root['mediaType']
        else:
            # A valid OCI layout may put platform descriptors directly in index.json.
            raw, manifests = index_raw, roots
            root_digest = 'sha256:' + hashlib.sha256(raw).hexdigest()
            root_type = index.get('mediaType', 'application/vnd.oci.image.index.v1+json')
            require(root_type in INDEX_TYPES, 'invalid direct OCI root index')
        require(isinstance(manifests, list) and len(manifests) == len(expected_platforms), 'incorrect OCI platform manifest count')
        targets, platforms = set(), []
        for image in manifests:
            member = descriptor(image, MANIFEST_TYPES)
            reached.add(image['digest'])
            manifest = json_member(member.name)
            require(manifest.get('schemaVersion') == 2 and manifest.get('mediaType', image['mediaType']) == image['mediaType'], 'invalid OCI image manifest')
            config = manifest.get('config')
            config_member = descriptor(config, CONFIG_TYPES)
            reached.add(config['digest'])
            config_doc = json_member(config_member.name)
            target = (config_doc.get('os'), config_doc.get('architecture'))
            platform = image.get('platform')
            if platform is not None:
                require(isinstance(platform, dict) and (platform.get('os'), platform.get('architecture')) == target, 'OCI config differs from platform descriptor')
                require(not platform.get('variant') or (target[1] == 'arm64' and platform['variant'] == 'v8'), 'unsupported OCI platform variant')
            # OCI platform descriptors are optional. The hashed config is the
            # authoritative platform when BuildKit omits this annotation.
            require(target in expected_platforms and target not in targets, 'unexpected or duplicate OCI platform')
            targets.add(target)
            layers = manifest.get('layers')
            require(isinstance(layers, list) and layers, 'OCI image has no layers')
            for layer in layers:
                descriptor(layer, LAYER_TYPES)
                reached.add(layer['digest'])
            platforms.append({'os': target[0], 'arch': target[1], 'manifest_digest': image['digest'], 'manifest_size': image['size'],
                              'manifest_media_type': image['mediaType'], 'config_digest': config['digest'],
                              'layers': [{'digest': layer['digest'], 'size': layer['size'], 'media_type': layer['mediaType']} for layer in layers]})
        require(targets == expected_platforms, 'missing OCI platform')
        if 'manifest.json' in members:
            # BuildKit also writes Docker compatibility metadata. It must name
            # exactly the verified configs/layers so scanners see the same graph.
            docker = json.loads(bytes_member(members['manifest.json']), object_pairs_hook=unique_object)
            require(isinstance(docker, list) and len(docker) == len(platforms), 'invalid Docker compatibility manifest')
            expected = {(p['config_digest'], tuple(layer['digest'] for layer in p['layers'])) for p in platforms}
            actual = set()
            for entry in docker:
                require(isinstance(entry, dict) and isinstance(entry.get('Layers'), list), 'invalid Docker compatibility entry')
                files = [entry.get('Config')] + entry['Layers']
                require(all(isinstance(name, str) and re.fullmatch(r'blobs/sha256/[0-9a-f]{64}', name) for name in files), 'unsafe Docker compatibility blob path')
                subject = ('sha256:' + files[0].rsplit('/', 1)[1], tuple('sha256:' + name.rsplit('/', 1)[1] for name in files[1:]))
                require(subject not in actual and subject in expected, 'Docker compatibility graph differs from OCI graph')
                actual.add(subject)
            require(actual == expected, 'Docker compatibility graph lacks a platform')
        require(reached == set(blobs), 'OCI archive contains unreferenced blobs')
        report = {'schema_version': 1, 'status': 'passed', 'scope': 'complete-oci-platform-graph', 'archive_sha256': sha256(archive),
                  'root_digest': root_digest, 'root_media_type': root_type, 'platforms': sorted(platforms, key=lambda p: p['arch'])}
        return raw, report


def main():
    archive, output = Path(sys.argv[1]), Path(sys.argv[2])
    raw, report = inspect(archive)
    output.mkdir(parents=True, exist_ok=True)
    (output / 'image-index.json').write_bytes(raw)
    (output / 'image-digest.txt').write_text(report['root_digest'] + '\n')
    (output / 'image-platforms.json').write_text(json.dumps(report, indent=2, sort_keys=True) + '\n')
    print('inspect-image-archive: verified complete Linux AMD64/ARM64 descriptor graph')


if __name__ == '__main__':
    main()
