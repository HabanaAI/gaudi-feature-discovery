# Copyright (c) 2022, HabanaLabs Ltd.  All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# -- Build stage
ARG HFD_VERSION

FROM golang:1.25.8 AS golang
FROM gcr.io/distroless/static:nonroot as distroless

FROM golang AS workspace
ARG HTTP_PROXY=""
ARG HTTPS_PROXY=""
ARG NO_PROXY=""
ENV http_proxy=${HTTP_PROXY}
ENV https_proxy=${HTTPS_PROXY}
ENV no_proxy=${NO_PROXY}
ENV HTTP_PROXY=${HTTP_PROXY}
ENV HTTPS_PROXY=${HTTPS_PROXY}
ENV NO_PROXY=${NO_PROXY}
ARG REMOTE_USER=user-name-goes-here
ARG REMOTE_UID=1000
ARG REMOTE_GID=$REMOTE_UID
RUN getent group ${REMOTE_GID} > /dev/null 2>&1 || groupadd --gid ${REMOTE_GID} ${REMOTE_USER}
RUN useradd --uid $REMOTE_UID --gid $REMOTE_GID -m $REMOTE_USER

FROM golang AS builder
WORKDIR /workspace
COPY go.mod go.sum ./
RUN go mod download && \
    go get github.com/google/go-licenses

COPY . ./
ENV GCFLAGS="all=-spectre=all -N -l"
ENV ASMFLAGS="all=-spectre=all"
ENV LDFLAGS="all=-s -w"
ENV GOFLAGS=""
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build ${GOFLAGS} -trimpath -mod=readonly -gcflags="${GCFLAGS}" -asmflags="${ASMFLAGS}" -ldflags="${LDFLAGS}" -a -o hfd ./cmd/
RUN mkdir licenses && go run github.com/google/go-licenses save ./... --save_path=licenses || true

# Verify binary build specs with checksec
ENV CHECKSEC_REF="No RELRO,No Canary found,NX enabled,No PIE,N/A,N/A,No Symbols,N/A,0,0"
RUN apt-get update -y && apt-get --no-install-recommends -y install file && \
    wget -q https://raw.githubusercontent.com/slimm609/checksec/refs/heads/main/checksec.bash -O checksec && \
    chmod +x checksec && \
    ./checksec --file=/workspace/hfd --output=csv | grep -q "${CHECKSEC_REF}"

# -- Deployment image
FROM distroless as production
ARG HFD_VERSION=""
ARG BUILD_DATE=""
WORKDIR /
COPY --from=builder /workspace/hfd /hfd
COPY --from=builder /workspace/licenses /licenses
COPY LICENSE /licenses/intel-gaudi-feature-discovery/LICENSE
USER 65532:65532
ENTRYPOINT ["/hfd"]

LABEL org.opencontainers.image.vendor="Intel Corp."
LABEL org.opencontainers.image.title="Habana Feature Discovery (HFD)"
LABEL org.opencontainers.image.description="Habana Feature Discovery (HFD) is a utility that detects Intel Gaudi devices and their features."
LABEL org.opencontainers.image.version="${HFD_VERSION}"
LABEL org.opencontainers.image.created="${BUILD_DATE}"
LABEL org.opencontainers.image.source=https://github.com/HabanaAI/gaudi-feature-discovery
LABEL org.opencontainers.image.licenses="Apache-2.0"
