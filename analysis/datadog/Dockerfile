# Copyright 2025 Google LLC
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#     https://www.apache.org/licenses/LICENSE-2.0
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
ARG GO_VERSION=1.25

FROM golang:${GO_VERSION} AS go-build
WORKDIR /app
COPY go.mod go.sum ./
COPY packages/ ./packages/
COPY analysis/datadog/ ./analysis/datadog/
COPY analysis/util/ ./analysis/util/
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /datadog-analysis  ./analysis/datadog

FROM gcr.io/distroless/static-debian12:latest
COPY --from=go-build /datadog-analysis /datadog-analysis
ENTRYPOINT ["/datadog-analysis"]