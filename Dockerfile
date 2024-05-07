FROM alpine:latest

WORKDIR /app
ADD revcat .
RUN mkdir ./cache
RUN mkdir /opt/revcat
RUN mkdir ./data
RUN mkdir ./tools
ADD ./config/revcat.toml /opt/revcat/revcat.toml
COPY data/ ./data/
RUN ls -la ./data/*
COPY tools/ ./tools/
RUN ls -la ./tools/*
ENTRYPOINT ["./primobridge", "-cfg", "/opt/primobridge/primobridge.toml"]
EXPOSE 8443
