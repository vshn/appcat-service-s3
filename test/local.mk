setup_envtest_bin = $(kind_dir)/setup-envtest
crossplane_sentinel = $(kind_dir)/crossplane-setup-sentinel

# Prepare binary
# We need to set the Go arch since the binary is meant for the user's OS.
$(setup_envtest_bin): export GOOS = $(shell go env GOOS)
$(setup_envtest_bin): export GOARCH = $(shell go env GOARCH)
$(setup_envtest_bin):
	@mkdir -p $(kind_dir)
	cd test && go build -o $@ sigs.k8s.io/controller-runtime/tools/setup-envtest
	$@ $(ENVTEST_ADDITIONAL_FLAGS) use '$(ENVTEST_K8S_VERSION)!'
	chmod -R +w $(kind_dir)/k8s

.PHONY: local-install
local-install: export KUBECONFIG = $(KIND_KUBECONFIG)
local-install: kind-load-image install-crd $(kind_dir)/.credentials.yaml ## Install Operator in local cluster
	helm upgrade --install provider-cloudscale charts/provider-cloudscale \
		--create-namespace --namespace provider-cloudscale-system \
		--set "operator.args[0]=--log-level=1" \
		--set "operator.args[1]=operator" \
		--set podAnnotations.date="$(shell date)" \
		--wait $(local_install_args)
	kubectl apply -n provider-cloudscale-system -f $(kind_dir)/.credentials.yaml

.PHONY: crossplane-setup
crossplane-setup: $(crossplane_sentinel)

$(crossplane_sentinel): export KUBECONFIG = $(KIND_KUBECONFIG)
$(crossplane_sentinel): local-install
	helm repo add crossplane https://charts.crossplane.io/stable
	helm upgrade --install crossplane crossplane/crossplane \
		--create-namespace \
		--namespace crossplane-system \
		--set "args[0]='--debug'" \
		--set "args[1]='--enable-composition-revisions'" \
		--wait
	kubectl create clusterrolebinding crossplane:cluster-admin --clusterrole cluster-admin --serviceaccount crossplane-system:crossplane --dry-run=true -oyaml | kubectl apply -f -
	@touch $@

.PHONY: crossplane-composition
crossplane-composition: export KUBECONFIG = $(KIND_KUBECONFIG)
crossplane-composition: crossplane-setup local-install
	kubectl apply -f appcat/objectbucket/xrd.yaml
	kubectl apply -f appcat/objectbucket/composition.yaml

.PHONY: kind-run-operator
kind-run-operator: export KUBECONFIG = $(KIND_KUBECONFIG)
kind-run-operator: kind-setup ## Run in Operator mode against kind cluster (you may also need `install-crd`)
	go run . -v 1 operator

###
### Integration Tests
###

.PHONY: test-integration
test-integration: export ENVTEST_CRD_DIR = $(shell realpath $(envtest_crd_dir))
test-integration: $(setup_envtest_bin) .envtest_crds ## Run integration tests against code
	export KUBEBUILDER_ASSETS="$$($(setup_envtest_bin) $(ENVTEST_ADDITIONAL_FLAGS) use -i -p path '$(ENVTEST_K8S_VERSION)!')" && \
	go test -tags=integration -coverprofile cover.out -covermode atomic ./...

test-composition: export KUBECONFIG = $(KIND_KUBECONFIG)
test-composition: crossplane-composition ## run tests with kuttl NOTE: no other instance should be provisioned when running the tests!
	kubectl-kuttl test ./test

envtest_crd_dir ?= $(kind_dir)/crds

.envtest_crd_dir:
	@mkdir -p $(envtest_crd_dir)
	@cp -r package/crds $(kind_dir)

.envtest_crds: .envtest_crd_dir

$(kind_dir)/.credentials.yaml:
	kubectl create secret generic --from-literal CLOUDSCALE_API_TOKEN=$(shell echo $$CLOUDSCALE_API_TOKEN) -o yaml --dry-run=client api-token > $@
