VERSION=`git describe --tags`
BUILDTIME=`date -u +%a,\ %d\ %b\ %Y\ %H:%M:%S\ GMT`
LDFLAGS=""
BINARY=aragorn

all:
	go build ${LDFLAGS} -o ${BINARY} .

dev:
	go fmt . ./lib/...
	go vet . ./lib/...
	go test . ./lib/...

install: dev
	go install .

release: dev
	GOOS=linux GOARCH=amd64 go build -o ${BINARY}_linux ${LDFLAGS} .
	GOOS=windows GOARCH=amd64 go build -o ${BINARY}_windows ${LDFLAGS} .
	GOOS=darwin GOARCH=amd64 go build -o ${BINARY}_darwin ${LDFLAGS} .

clean:
	if [ -f ${BINARY} ] ; then rm ${BINARY} ; fi
	if [ -f ${BINARY}_linux ] ; then rm ${BINARY}_linux ; fi
	if [ -f ${BINARY}_windows ] ; then rm ${BINARY}_windows ; fi
	if [ -f ${BINARY}_darwin ] ; then rm ${BINARY}_darwin ; fi
