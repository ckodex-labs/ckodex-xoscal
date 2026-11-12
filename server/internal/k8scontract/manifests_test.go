package k8scontract

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestBetaManifestsKeepSingleWriterAndSecretBackedRuntime(t *testing.T) {
	root := repositoryRoot(t)
	deployment := documentByKind(t, filepath.Join(root, "k8s/server/deployment.yaml"), "Deployment")
	configMap := documentByKind(t, filepath.Join(root, "k8s/server/configmap.yaml"), "ConfigMap")
	pvc := documentByKind(t, filepath.Join(root, "k8s/server/deployment.yaml"), "PersistentVolumeClaim")

	deploymentSpec := requiredMap(t, deployment, "spec")
	if got := requiredInt(t, deploymentSpec, "replicas"); got != 1 {
		t.Fatalf("beta deployment must keep one SQLite writer, got replicas=%d", got)
	}

	template := requiredMap(t, requiredMap(t, deploymentSpec, "template"), "spec")
	containers := requiredSlice(t, template, "containers")
	if len(containers) != 1 {
		t.Fatalf("beta deployment must have one server container, got %d", len(containers))
	}
	container := requiredMapValue(t, containers[0], "server container")
	if got := requiredString(t, container, "image"); got != "ghcr.io/ckodex-labs/ckodex-xoscal" {
		t.Fatalf("server container image = %q, want ghcr.io/ckodex-labs/ckodex-xoscal", got)
	}
	assertEnv(t, container, "XOSCAL_STORE_DSN", "/data/oscal.db")
	assertArgPair(t, container, "-config", "/etc/xoscal/config.yaml")

	assertSecretMount(t, container, "tls", "/etc/xoscal/tls")
	assertSecretMount(t, container, "auth", "/etc/xoscal/auth")
	assertVolumeSource(t, template, "data", "persistentVolumeClaim", "claimName", "xoscal-server-data")
	assertVolumeSource(t, template, "config", "configMap", "name", "xoscal-server-config")
	assertVolumeSource(t, template, "tls", "secret", "secretName", "xoscal-server-tls")
	assertVolumeSource(t, template, "auth", "secret", "secretName", "xoscal-server-auth")

	configData := requiredMap(t, configMap, "data")
	configText := requiredString(t, configData, "config.yaml")
	var runtimeConfig map[string]any
	if err := yaml.Unmarshal([]byte(configText), &runtimeConfig); err != nil {
		t.Fatalf("decode embedded server config: %v", err)
	}
	serverConfig := requiredMap(t, runtimeConfig, "server")
	if got := requiredBool(t, serverConfig, "require_tls"); !got {
		t.Fatal("server config must require TLS")
	}
	if got := requiredString(t, serverConfig, "tls_cert_path"); got != "/etc/xoscal/tls/tls.crt" {
		t.Fatalf("unexpected TLS certificate path %q", got)
	}
	if got := requiredString(t, serverConfig, "tls_key_path"); got != "/etc/xoscal/tls/tls.key" {
		t.Fatalf("unexpected TLS key path %q", got)
	}
	securityConfig := requiredMap(t, runtimeConfig, "security")
	if got := requiredString(t, securityConfig, "auth_mode"); got != "token" {
		t.Fatalf("beta server must use token auth, got %q", got)
	}
	if got := requiredString(t, securityConfig, "auth_token_file"); got != "/etc/xoscal/auth/token" {
		t.Fatalf("unexpected token path %q", got)
	}

	pvcSpec := requiredMap(t, pvc, "spec")
	accessModes := requiredSlice(t, pvcSpec, "accessModes")
	if len(accessModes) != 1 || accessModes[0] != "ReadWriteOnce" {
		t.Fatalf("beta SQLite volume must be ReadWriteOnce, got %#v", accessModes)
	}
	resources := requiredMap(t, pvcSpec, "resources")
	requests := requiredMap(t, resources, "requests")
	if got := requiredString(t, requests, "storage"); got == "" {
		t.Fatal("beta PVC must declare a storage request")
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate manifest test")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(source), "..", "..", ".."))
}

func documentByKind(t *testing.T, path, kind string) map[string]any {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer func() { _ = file.Close() }()

	decoder := yaml.NewDecoder(file)
	for {
		var document map[string]any
		if err := decoder.Decode(&document); err != nil {
			if err.Error() == "EOF" {
				break
			}
			t.Fatalf("decode %s: %v", path, err)
		}
		if len(document) == 0 {
			continue
		}
		if value, ok := document["kind"].(string); ok && value == kind {
			return document
		}
	}
	t.Fatalf("%s does not contain a %s document", path, kind)
	return nil
}

func requiredMap(t *testing.T, parent map[string]any, key string) map[string]any {
	t.Helper()
	value, ok := parent[key].(map[string]any)
	if !ok {
		t.Fatalf("%s must be an object, got %#v", key, parent[key])
	}
	return value
}

func requiredMapValue(t *testing.T, value any, context string) map[string]any {
	t.Helper()
	result, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("%s must be an object, got %#v", context, value)
	}
	return result
}

func requiredSlice(t *testing.T, parent map[string]any, key string) []any {
	t.Helper()
	value, ok := parent[key].([]any)
	if !ok {
		t.Fatalf("%s must be an array, got %#v", key, parent[key])
	}
	return value
}

func requiredString(t *testing.T, parent map[string]any, key string) string {
	t.Helper()
	value, ok := parent[key].(string)
	if !ok {
		t.Fatalf("%s must be a string, got %#v", key, parent[key])
	}
	return value
}

func requiredInt(t *testing.T, parent map[string]any, key string) int {
	t.Helper()
	value, ok := parent[key].(int)
	if !ok {
		t.Fatalf("%s must be an integer, got %#v", key, parent[key])
	}
	return value
}

func requiredBool(t *testing.T, parent map[string]any, key string) bool {
	t.Helper()
	value, ok := parent[key].(bool)
	if !ok {
		t.Fatalf("%s must be a boolean, got %#v", key, parent[key])
	}
	return value
}

func assertEnv(t *testing.T, container map[string]any, name, expected string) {
	t.Helper()
	for _, item := range requiredSlice(t, container, "env") {
		env := requiredMapValue(t, item, "environment entry")
		if requiredString(t, env, "name") == name {
			if got := requiredString(t, env, "value"); got != expected {
				t.Fatalf("environment %s=%q, want %q", name, got, expected)
			}
			return
		}
	}
	t.Fatalf("environment variable %s is missing", name)
}

func assertArgPair(t *testing.T, container map[string]any, flag, expected string) {
	t.Helper()
	args := requiredSlice(t, container, "args")
	for index, value := range args {
		if value == flag && index+1 < len(args) && args[index+1] == expected {
			return
		}
	}
	t.Fatalf("container args must include %s %s", flag, expected)
}

func assertSecretMount(t *testing.T, container map[string]any, name, path string) {
	t.Helper()
	for _, item := range requiredSlice(t, container, "volumeMounts") {
		mount := requiredMapValue(t, item, "volume mount")
		if requiredString(t, mount, "name") == name {
			if got := requiredString(t, mount, "mountPath"); got != path {
				t.Fatalf("secret mount %s uses %q, want %q", name, got, path)
			}
			if readOnly, ok := mount["readOnly"].(bool); !ok || !readOnly {
				t.Fatalf("secret mount %s must be read-only", name)
			}
			return
		}
	}
	t.Fatalf("secret mount %s is missing", name)
}

func assertVolumeSource(t *testing.T, podSpec map[string]any, name, source, key, expected string) {
	t.Helper()
	for _, item := range requiredSlice(t, podSpec, "volumes") {
		volume := requiredMapValue(t, item, "volume")
		if requiredString(t, volume, "name") != name {
			continue
		}
		volumeSource := requiredMap(t, volume, source)
		if got := requiredString(t, volumeSource, key); got != expected {
			t.Fatalf("volume %s %s=%q, want %q", name, key, got, expected)
		}
		return
	}
	t.Fatalf("volume %s is missing", name)
}

func TestKubernetesDeploymentsRespectBetaConstraints(t *testing.T) {
	root := repositoryRoot(t)
	k8sDir := filepath.Join(root, "k8s")
	if _, err := os.Stat(filepath.Join(k8sDir, "spire-sidecar.yaml")); err == nil {
		t.Fatal("k8s/spire-sidecar.yaml must not exist while SPIRE auth is unimplemented")
	}
	if _, err := os.Stat(filepath.Join(k8sDir, "server/kustomization.yaml")); err != nil {
		t.Fatalf("k8s/server/kustomization.yaml missing: %v", err)
	}
	err := filepath.WalkDir(k8sDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || (filepath.Ext(path) != ".yaml" && filepath.Ext(path) != ".yml") {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(raw), "spire-auth-config") && filepath.Base(path) != "spire-auth-configmap.yaml" {
			t.Fatalf("%s references spire-auth-config while SPIRE auth is unimplemented", path)
		}
		dec := yaml.NewDecoder(strings.NewReader(string(raw)))
		for {
			var doc map[string]any
			if err := dec.Decode(&doc); err != nil {
				if err.Error() == "EOF" {
					break
				}
				return err
			}
			if len(doc) == 0 {
				continue
			}
			if kind, _ := doc["kind"].(string); kind == "Deployment" {
				spec := requiredMap(t, doc, "spec")
				if got := requiredInt(t, spec, "replicas"); got != 1 {
					t.Fatalf("%s Deployment replicas=%d, must be 1 for SQLite", path, got)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk k8s manifests: %v", err)
	}
}
