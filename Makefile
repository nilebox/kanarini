OS = $$(uname -s | tr A-Z a-z)
BINARY_PREFIX_DIRECTORY=$(OS)_amd64_stripped

.PHONY: setup
setup:
	dep ensure

.PHONY: fmt-bazel
fmt-bazel:
	bazel run //:buildozer
	-bazel run //:buildifier

.PHONY: update-bazel
update-bazel:
	bazel run //:gazelle --verbose_failures

.PHONY: fmt
fmt:
	bazel run //:goimports

.PHONY: test
test: fmt update-bazel \
	bazel test \
		--test_env=KUBE_PATCH_CONVERSION_DETECTOR=true \
		--test_env=KUBE_CACHE_MUTATION_DETECTOR=true \
		-- //... -//vendor/...

.PHONY: verify
verify:
	bazel run //:buildifier_check
	VERIFY_CODE=--verify-only make generate

.PHONY: lint
lint:
	bazel run //:gometalinter

.PHONY: generate
generate: generate-client generate-deepcopy

.PHONY: generate-client
generate-client:
	bazel build //vendor/k8s.io/code-generator/cmd/client-gen
	# Generate the versioned clientset (pkg/client/clientset_generated/clientset)
	bazel-bin/vendor/k8s.io/code-generator/cmd/client-gen/$(BINARY_PREFIX_DIRECTORY)/client-gen $(VERIFY_CODE) \
	--input-base "github.com/nilebox/kanarini/pkg/apis/" \
	--input "kanarini/v1alpha1" \
	--clientset-path "github.com/nilebox/kanarini/pkg/client/clientset_generated/" \
	--clientset-name "clientset" \
	--go-header-file "build/code-generator/boilerplate.go.txt"

.PHONY: generate-deepcopy
generate-deepcopy:
	bazel build //vendor/k8s.io/code-generator/cmd/deepcopy-gen
	# Generate deep copies
	bazel-bin/vendor/k8s.io/code-generator/cmd/deepcopy-gen/$(BINARY_PREFIX_DIRECTORY)/deepcopy-gen $(VERIFY_CODE) \
	--go-header-file "build/code-generator/boilerplate.go.txt" \
	--input-dirs "github.com/nilebox/kanarini/pkg/apis/kanarini/v1alpha1" \
	--bounding-dirs "github.com/nilebox/kanarini/pkg/apis/kanarini/v1alpha1" \
	--output-file-base zz_generated.deepcopy

.PHONY: docker-kanarini
docker-kanarini:
	bazel build \
		--platforms=@io_bazel_rules_go//go/toolchain:linux_amd64 \
		//cmd/kanarini:container

# Export docker image into local Docker
.PHONY: docker-export-kanarini
docker-export-kanarini:
	bazel run \
		--platforms=@io_bazel_rules_go//go/toolchain:linux_amd64 \
		//cmd/kanarini:container \
		-- \
		--norun

.PHONY: release-kanarini
release-kanarini: update-bazel
	bazel run \
		--platforms=@io_bazel_rules_go//go/toolchain:linux_amd64 \
		//cmd/kanarini:push_docker

.PHONY: docker-example
docker-example:
	bazel build \
		--platforms=@io_bazel_rules_go//go/toolchain:linux_amd64 \
		//cmd/example:container

# Export docker image into local Docker
.PHONY: docker-export-example
docker-export-example:
	bazel run \
		--platforms=@io_bazel_rules_go//go/toolchain:linux_amd64 \
		//cmd/example:container \
		-- \
		--norun

.PHONY: release-example
release-example: update-bazel
	bazel run \
		--platforms=@io_bazel_rules_go//go/toolchain:linux_amd64 \
		//cmd/example:push_docker
