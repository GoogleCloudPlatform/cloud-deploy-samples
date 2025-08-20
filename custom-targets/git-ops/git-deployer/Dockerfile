# Copyright 2023 Google LLC

# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at

#     https://www.apache.org/licenses/LICENSE-2.0

# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

ARG GCLOUD_VERSION=456.0.0
ARG GO_VERSION=1.24

FROM golang:${GO_VERSION} AS go-build
ARG COMMIT_SHA=unknown
WORKDIR /app

COPY go.mod go.sum ./
COPY packages/ ./packages/
COPY custom-targets/git-ops/git-deployer/*.go ./custom-targets/git-ops/git-deployer/
COPY custom-targets/git-ops/git-deployer/providers/*.go ./custom-targets/git-ops/git-deployer/providers/
RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-X github.com/GoogleCloudPlatform/cloud-deploy-samples/custom-targets/util/clouddeploy.GitCommit=${COMMIT_SHA}" -o /git-deployer ./custom-targets/git-ops/git-deployer

FROM  gcr.io/google.com/cloudsdktool/google-cloud-cli:${GCLOUD_VERSION} AS release
COPY --from=go-build /git-deployer /bin/git-deployer

CMD ["/bin/git-deployer"]