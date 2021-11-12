FROM techknowlogick/xgo:latest AS base
# custom part to add go releaser to the image
ENV GORELEASER_DOWNLOAD_FILE=goreleaser_Linux_x86_64.tar.gz

RUN curl -s https://api.github.com/repos/goreleaser/goreleaser/releases/latest \
    | grep "browser_download_url.*${GORELEASER_DOWNLOAD_FILE}\"" \
    | cut -d : -f 2,3 \
    | tr -d \" \
    | wget -qi -

RUN  tar -xzf ${GORELEASER_DOWNLOAD_FILE} -C /usr/bin/ goreleaser \
        && rm ${GORELEASER_DOWNLOAD_FILE};

# overwrite entrypoint from parent docker image
ENTRYPOINT ["/usr/bin/env"]

# print goreleaser version by default
CMD ["goreleaser", "-v"]