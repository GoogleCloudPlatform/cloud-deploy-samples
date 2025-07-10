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

# Copy the model-deployer module's go.mod and go.sum
COPY custom-targets/vertex-ai/model-deployer/go.mod ./custom-targets/vertex-ai/model-deployer/
COPY custom-targets/vertex-ai/model-deployer/go.sum ./custom-targets/vertex-ai/model-deployer/
COPY custom-targets/vertex-ai/model-deployer/*.go ./custom-targets/vertex-ai/model-deployer/
# Copy the util module's go.mod and go.sum
COPY custom-targets/util/go.mod ./custom-targets/util/
COPY custom-targets/util/go.sum ./custom-targets/util/
COPY custom-targets/util/** ./custom-targets/util/
RUN ls -R
# Change to the model-deployer module directory to run go mod download
# The replace directive in model-deployer's go.mod points to ../../util
# Relative to /app/custom-targets/vertex-ai/model-deployer, this means /app/custom-targets/util
WORKDIR /app/custom-targets/vertex-ai/model-deployer
RUN go mod tidy
# Download dependencies (including the replaced local util module)
RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-X github.com/GoogleCloudPlatform/cloud-deploy-samples/custom-targets/util/clouddeploy.GitCommit=${COMMIT_SHA}" -o /vertex-ai-deployer

FROM gcr.io/distroless/static-debian12:latest AS release
COPY --from=go-build /vertex-ai-deployer /bin/vertex-ai-deployer

CMD ["/bin/vertex-ai-deployer"]