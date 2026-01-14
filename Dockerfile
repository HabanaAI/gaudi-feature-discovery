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

FROM golang:1.25.4 AS golang
FROM gcr.io/distroless/static:nonroot as distroless



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
CMD ["/hfd"]

LABEL org.opencontainers.image.vendor="Intel Corp."
LABEL org.opencontainers.image.title="Habana Feature Discovery (HFD)"
LABEL org.opencontainers.image.description="Habana Feature Discovery (HFD) is a utility that detects Intel Gaudi devices and their features."
LABEL org.opencontainers.image.version="${HFD_VERSION}"
LABEL org.opencontainers.image.created="${BUILD_DATE}"
LABEL org.opencontainers.image.source=https://github.com/HabanaAI/gaudi-feature-discovery
LABEL org.opencontainers.image.licenses="Apache-2.0"
