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

ARG GO_VERSION=1.24

FROM golang:${GO_VERSION} AS go-build
ARG COMMIT_SHA=unknown
WORKDIR /app
COPY go.mod go.sum ./
COPY custom-targets/vertex-ai-pipeline/pipeline-deployer/*.go ./custom-targets/vertex-ai-pipeline/pipeline-deployer/
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-X github.com/GoogleCloudPlatform/cloud-deploy-samples/custom-targets/util/clouddeploy.GitCommit=${COMMIT_SHA}" -o /vertex-ai-deployer ./custom-targets/vertex-ai-pipeline/pipeline-deployer/


FROM gcr.io/distroless/static-debian12:latest AS release
COPY --from=go-build /vertex-ai-deployer /bin/vertex-ai-deployer

CMD ["/bin/vertex-ai-deployer"]