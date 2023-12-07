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

FROM golang:1.21

ARG commit_sha

ENV COMMIT_SHA=$commit_sha

WORKDIR /app
COPY go.mod go.sum ./
COPY ./*.go ./
COPY clouddeploy/go.mod ./clouddeploy/
COPY clouddeploy/*.go ./clouddeploy/

RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-X github.com/GoogleCloudPlatform/cloud-deploy-samples/custom-targets/util/clouddeploy.GitCommit=$COMMIT_SHA" -o /cd-ai-deployer

CMD ["/cd-ai-deployer"]