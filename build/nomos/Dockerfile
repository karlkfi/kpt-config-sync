FROM k8s.gcr.io/debian-base:0.4.0

# https://github.com/GoogleCloudPlatform/google-cloud-go/issues/791#issuecomment-353689746
RUN apt-get update && apt-get install -y \
  ca-certificates

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

CMD echo -e "Need to specify a Nomos binary:\n\n$(ls .)" && exit 1
