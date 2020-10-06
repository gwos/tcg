REGISTRY_REPO   := groundworkdevelopment/$$(basename $$PWD)
DOCKER_REGISTRY := docker.io
COMMIT_HASH     := $$(git log -1 --pretty=%h)

# Branch name will be in either TRAVIS_BRANCH or TRAVIS_PULL_REQUEST_BRANCH,
# depending on whether the build is a branch build or pull request build
BRANCH          := ${TRAVIS_PULL_REQUEST_BRANCH}
ifeq ($(BRANCH),)
    BRANCH      := ${TRAVIS_BRANCH}
endif
ifeq ($(BRANCH),)
    BRANCH      := unknown
endif
# Replace / with - in branch name so it's a valid docker tag
ESCAPED_BRANCH  := $(subst /,-,$(BRANCH))

IMG             := ${REGISTRY_REPO}:${ESCAPED_BRANCH}
BUILD_ARGS      := ${BUILD_ARGS} \
                    --build-arg COMMIT_HASH \
                    --build-arg BRANCH \
                    --build-arg TRAVIS_BUILD_ID \
                    --build-arg TRAVIS_COMMIT \
                    --build-arg TRAVIS_COMMIT_MESSAGE \
                    --build-arg TRAVIS_JOB_ID \
                    --build-arg TRAVIS_JOB_WEB_URL \
                    --build-arg TRAVIS_TAG

all: echo login build tag push


echo:
	@echo =======================================
	@echo REGISTRY_REPO:       ${REGISTRY_REPO}
	@echo COMMIT_HASH:         ${COMMIT_HASH}
	@echo IMG:                 ${IMG}
	@echo TRAVIS_TAG:          ${TRAVIS_TAG}
	@echo BRANCH:              ${BRANCH}
	@echo ESCAPED_BRANCH:      ${ESCAPED_BRANCH}
	@echo DOCKER_REGISTRY:     ${DOCKER_REGISTRY}
	@echo DOCKER_HUB_USERNAME: ${DOCKER_HUB_USERNAME}
	@echo BUILD_ARGS:          ${BUILD_ARGS}
	@echo =======================================

all: echo login build tag push trigger-docker-hub-build

login:
	echo "$${DOCKER_HUB_PASSWORD}" | docker login -u "$${DOCKER_HUB_USERNAME}" --password-stdin "$${DOCKER_REGISTRY}"


build:
	docker build ${BUILD_ARGS} -t ${IMG} .


tag:
	# If the current build is for a git tag, TRAVIS_TAG will be set to the tag's name.
    ifeq ($(TRAVIS_TAG),)
			@echo "Current build does not correspond to a git tag (TRAVIS_TAG is empty); skipping"
    else
			@echo "Current build corresponds to git tag $(TRAVIS_TAG); tagging docker image..."
			docker tag ${IMG} ${REGISTRY_REPO}:${TRAVIS_TAG}
    endif

	# If branch is master, tag latest
    ifeq ($(BRANCH),master)
			docker tag ${IMG} ${REGISTRY_REPO}:latest
    endif


push:
	./script/docker-push ${REGISTRY_REPO} ${DOCKER_REGISTRY}


trigger-docker-hub-build:
	# trigger docker hub build if master
    ifeq ($(BRANCH),master)
    ifeq ($(DOCKER_HUB_TRIGGER_URL),)
		@echo "DOCKER_HUB_TRIGGER_URL is empty; skipping"
    else
		@echo "DOCKER_HUB_TRIGGER_URL is set to $(DOCKER_HUB_TRIGGER_URL); triggering NAGIOS build..."
		curl -i -H "Content-Type: application/json" -H "Accept: application/json" -H "Travis-API-Version: 3" -H "Authorization: token ${TRAVIS_ACCESS_TOKEN}" --data '{"request": {"branch": "master"}}' -X POST "${DOCKER_HUB_TRIGGER_URL}"
    endif
    endif
