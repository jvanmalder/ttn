FROM alpine
RUN apk --update --no-cache add ca-certificates
ADD ./release/ttn-linux-amd64 /usr/local/bin/ttn
RUN chmod 755 /usr/local/bin/ttn
# FIX FOR GO COMPILED WITH GLIBC
RUN mkdir /lib64 && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2
ENTRYPOINT ["/usr/local/bin/ttn"]
