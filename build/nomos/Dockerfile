FROM k8s.gcr.io/debian-base:1.0.0

# https://github.com/GoogleCloudPlatform/google-cloud-go/issues/791#issuecomment-353689746
RUN apt-get update && apt-get install -y \
  ca-certificates libcap2-bin bash

RUN mkdir -p /opt/nomos/bin
ENV PATH="/opt/nomos/bin:${PATH}"
WORKDIR /opt/nomos/bin

# Nomos binaries.
COPY git-importer git-importer
COPY monitor monitor
COPY nomos nomos
COPY resourcequota-admission-controller resourcequota-admission-controller
COPY syncer syncer

# License file required for on-prem release.
COPY LICENSE LICENSE

# Required by admission controllers.
EXPOSE 443

# Set capabilities on nomos binaries. Note that we have to do this here rather than in a Pod spec
# due to https://github.com/kubernetes/kubernetes/issues/56374.
RUN setcap cap_net_bind_service+eip resourcequota-admission-controller

# Set up a HOME directory for non-root user
RUN mkdir -p /nomos
RUN chown 1000 /nomos
ENV HOME="/nomos"

# Switch to non-root user
USER 1000

CMD echo -e "Need to specify a Nomos binary:\n\n$(ls .)" && exit 1
