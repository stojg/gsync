VERSION=`git describe --tags`
BUILDTIME=`date -u +%a,\ %d\ %b\ %Y\ %H:%M:%S\ GMT`
BINARY=gsync

all:
	go build -o ${BINARY} .

dev:
	go fmt . ./lib/...
	go vet . ./lib/...

install: dev
	go install .


test: dev
	go test . ./lib/...

release: dev
	GOOS=linux GOARCH=amd64 go build -o ${BINARY}_linux .

clean:
	if [ -f ${BINARY} ] ; then rm ${BINARY} ; fi
	if [ -f ${BINARY}_linux ] ; then rm ${BINARY}_linux ; fi
	if [ -f ${BINARY}_windows ] ; then rm ${BINARY}_windows ; fi
	if [ -f ${BINARY}_darwin ] ; then rm ${BINARY}_darwin ; fi
