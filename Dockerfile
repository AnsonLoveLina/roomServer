FROM golang
WORKDIR /go/src
ADD ./src .
RUN go get -u github.com/gpmgo/gopm
#RUN gopm get -g golang.org/x/tools
RUN gopm get -g golang.org/x/crypto
RUN gopm get -g golang.org/x/sys/unix
RUN gopm get -g github.com/stretchr/testify
RUN go get -v ./...
ENTRYPOINT ["roomServer"]