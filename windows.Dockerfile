# escape=`

FROM golang:1.11.0-windowsservercore AS build

ARG VERSION

WORKDIR C:\client
COPY . C:\client
RUN New-Item -ItemType directory -Path C:\build | Out-Null
RUN go build -ldflags '-X github.com/myopenfactory/client/cmd.version=$VERSION'

FROM golang:1.11.0-windowsservercore
COPY myOpenFactoryCA.crt C:\
RUN Import-Certificate -FilePath C:\myOpenFactoryCA.crt -CertStoreLocation cert:\LocalMachine\Root

WORKDIR C:\app\
COPY --from=build C:\client\client.exe C:\app\client.exe
CMD ["C:\app\client.exe"]