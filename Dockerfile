FROM alpine:latest

WORKDIR /app
ADD revcat .
RUN mkdir ./cache
RUN mkdir /opt/revcat
RUN mkdir ./data
RUN mkdir ./tools
COPY data/ ./data/
COPY tools/ ./tools/
ENTRYPOINT ["./revcat", "-config", "/opt/revcat/revcat.toml"]
EXPOSE 8443
