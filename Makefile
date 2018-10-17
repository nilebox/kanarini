OS = $$(uname -s | tr A-Z a-z)

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
