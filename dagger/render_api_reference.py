#!/usr/bin/env python3
"""Prebuild the complete human-readable API reference from generated OpenAPI.

The checked-in template and the release candidate use this same producer.
Scalar is an optional interactive view, never the producer of reference content.
"""
import argparse
import hashlib
import html
import json
from pathlib import Path
import re

START = '<!-- BEGIN GENERATED STATIC API REFERENCE -->'
END = '<!-- END GENERATED STATIC API REFERENCE -->'
METHODS = ('get', 'post', 'put', 'patch', 'delete', 'head', 'options', 'trace')
SERVICES = {f'oscal.services.v1.{name}' for name in (
    'OscalService', 'GovernanceService', 'TransparencyExchangeService', 'TransparencyGraphService')}


def escape(value):
    return html.escape(str(value), quote=True)


def anchor(identity):
    return 'api-operation-' + re.sub(r'[^a-z0-9]+', '-', identity.lower()).strip('-')


def schema_anchor(identity):
    # Definitions can differ only in case or punctuation; hashes preserve unique
    # stable anchors without changing their displayed canonical schema names.
    return 'api-schema-' + hashlib.sha256(identity.encode()).hexdigest()[:20]


def json_details(title, value, identity=None):
    return ('<details class="ck-stack--tight"><summary>' + escape(title) +
            '</summary><pre class="ck-code-block"' + (f' id="{escape(identity)}"' if identity else '') + '><code>' +
            escape(json.dumps(value, sort_keys=True, indent=2)) + '</code></pre></details>')


def operation_inventory(spec):
    operations = []
    seen = set()
    services = set()
    for path, item in sorted(spec.get('paths', {}).items()):
        for method in METHODS:
            if method not in item:
                continue
            operation = item[method]
            identity = operation.get('operationId')
            if not isinstance(identity, str) or not identity or identity in seen:
                raise ValueError('missing or duplicate static API operation identity')
            seen.add(identity)
            service = identity.rsplit('.', 1)[0]
            services.add(service)
            operations.append((service, path, method, operation))
    if services != SERVICES or not operations:
        raise ValueError('static API reference must cover all four generated services')
    if len({anchor(operation['operationId']) for _, _, _, operation in operations}) != len(operations):
        raise ValueError('static API operation anchor collision')
    return operations,services


def parameter_reference(parameter):
    if '$ref' in parameter:
        return ('<li><code>' + escape(parameter['$ref']) + '</code></li>')
    else:
        description = parameter.get('description', parameter.get('schema', {}).get('title', ''))
        required = 'required' if parameter.get('required') else 'optional'
        return ('<li><code>' + escape(parameter['name']) + '</code> (' +
                      escape(parameter['in']) + ', ' + required + ') — ' + escape(description) + '</li>')


def operation_reference(path,method,operation):
    output=[]
    output.append(f'<article class="ck-stack ck-stack--tight" data-api-operation="{escape(operation["operationId"])}" data-api-method="{method.upper()}" data-api-path="{escape(path)}">')
    output.append('<details class="ck-stack--tight" data-api-operation-details>')
    output.append(f'<summary><code>{method.upper()} {escape(path)}</code> — {escape(operation.get("summary", operation["operationId"]))}</summary>')
    # Fragment targets inside a disclosure expose their native ancestors
    # even without JavaScript; the stable ID does not change by view.
    output.append(f'<div class="ck-stack--tight" id="{anchor(operation["operationId"])}">')
    if operation.get('description'):
        output.append('<p>' + escape(operation['description']) + '</p>')
    output.append('<p class="ck-caption">Operation: <code>' + escape(operation['operationId']) + '</code></p>')
    if operation.get('parameters'):
        output.append('<ul aria-label="Request parameters">')
        for parameter in operation['parameters']:
            output.append(parameter_reference(parameter))
        output.append('</ul>')
    output.append('<p>Request body: ' + ('required' if operation.get('requestBody', {}).get('required') else
                  'optional' if operation.get('requestBody') else 'none') + '.</p>')
    output.append('<ul aria-label="Response status codes">')
    for status, response in sorted(operation.get('responses', {}).items()):
        output.append('<li><code>' + escape(status) + '</code> — ' + escape(response.get('description', '')) + '</li>')
    output.append('</ul>')
    output.append(json_details('Complete operation contract, request and response schemas', operation))
    output.append('</div></details></article>')
    return output


def service_reference(spec,service,operations):
    output=[]
    name = service.rsplit('.', 1)[-1]
    output.append(f'<div class="ck-stack" id="api-service-{escape(name.lower())}"><h3>{escape(name)}</h3>')
    tag = next((tag for tag in spec.get('tags', []) if tag.get('name') == service), {})
    if tag.get('description'):
        output.append('<p>' + escape(tag['description']) + '</p>')
    for owner, path, method, operation in operations:
        if owner != service:
            continue
        output.extend(operation_reference(path,method,operation))
    output.append('</div>')
    return output


def model_reference(schemas):
    output=[]
    output.append('<div id="api-model-schemas" class="ck-stack"><h3>Model schemas</h3><p>All local model definitions referenced by the operations are included here. Expand a definition to read its complete JSON schema without JavaScript.</p>')
    output.append(f'<details class="ck-stack--tight" id="api-model-definitions"><summary>Browse all {len(schemas)} model definitions</summary>')
    for name, definition in sorted(schemas.items()):
        output.append(f'<div data-api-schema="{escape(name)}">' + json_details(name, definition, schema_anchor(name)) + '</div>')
    output.append('</details></div></div>')
    return output


def reference(spec_bytes):
    spec = json.loads(spec_bytes)
    if not str(spec.get('openapi', '')).startswith('3.'):
        raise ValueError('static API producer requires generated OpenAPI 3')
    operations,services=operation_inventory(spec)
    schemas = spec.get('components', {}).get('schemas', {})
    if not schemas:
        raise ValueError('static API reference requires generated model schemas')
    output = [
        '<div id="static-api-reference" class="ck-stack" data-producer="Dagger/Openapi/render_api_reference.py"'
        f' data-openapi-sha256="{hashlib.sha256(spec_bytes).hexdigest()}" data-operation-count="{len(operations)}">',
        '<p>The complete reference below is generated before publication from the same OpenAPI contract '
        'as the interactive explorer. It remains available without JavaScript. Requests and responses use '
        'protobuf JSON envelopes; authentication depends on the deployment.</p>',
        '<nav aria-label="API service reference"><ul>',
    ]
    for service in sorted(services):
        output.append(f'<li><a href="#api-service-{escape(service.rsplit(".", 1)[-1].lower())}">{escape(service.rsplit(".", 1)[-1])}</a></li>')
    output.append('<li><a href="#api-model-schemas">Model schemas</a></li></ul></nav>')
    for service in sorted(services):
        output.extend(service_reference(spec,service,operations))
    output.extend(model_reference(schemas))
    return '\n'.join(output)

def render(document, spec_bytes):
    if document.count(START) != 1 or document.count(END) != 1 or document.index(START) >= document.index(END):
        raise ValueError('static API reference template marker drift')
    start = document.index(START) + len(START)
    end = document.index(END)
    return document[:start] + '\n' + reference(spec_bytes) + '\n' + document[end:]


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--root', type=Path, required=True)
    parser.add_argument('--check', action='store_true')
    args = parser.parse_args()
    document = args.root / 'docs.html'
    before = document.read_text()
    after = render(before, (args.root / 'openapi.json').read_bytes())
    if args.check:
        if before != after:
            raise ValueError('static API HTML differs from generated OpenAPI contract')
    else:
        document.write_text(after)
    print('complete static API reference: PASS')


if __name__ == '__main__':
    main()
