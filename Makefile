.PHONY: build docker docker-push release install test clean

VERSION := 0.1.7
DOCKER_IMAGE_NAME := jrhouston/tfk8s

build:
	go build -ldflags "-X main.toolVersion=${VERSION}"

docker:
	docker build -t ${DOCKER_IMAGE_NAME}:${VERSION} .

docker-push: docker
	docker push ${DOCKER_IMAGE_NAME}:${VERSION}

release: clean
	mkdir -p release/
	# FIXME use gox for this
	GOOS=linux GOOARCH=386 go build -ldflags "-X main.toolVersion=${VERSION}" -o release/tfk8s_${VERSION}_linux_386
	zip -j release/tfk8s_${VERSION}_linux_386.zip release/tfk8s_${VERSION}_linux_386
	GOOS=linux GOOARCH=amd64 go build -ldflags "-X main.toolVersion=${VERSION}" -o release/tfk8s_${VERSION}_linux_amd64
	zip -j release/tfk8s_${VERSION}_linux_amd64.zip release/tfk8s_${VERSION}_linux_amd64
	GOOS=linux GOOARCH=arm go build -ldflags "-X main.toolVersion=${VERSION}" -o release/tfk8s_${VERSION}_linux_arm
	zip -j release/tfk8s_${VERSION}_linux_arm.zip release/tfk8s_${VERSION}_linux_arm
	GOOS=darwin GOOARCH=amd64 go build -ldflags "-X main.toolVersion=${VERSION}" -o release/tfk8s_${VERSION}_darwin_amd64
	zip -j release/tfk8s_${VERSION}_darwin_amd64.zip release/tfk8s_${VERSION}_darwin_amd64
	GOOS=windows GOOARCH=amd64 go build -ldflags "-X main.toolVersion=${VERSION}" -o release/tfk8s_${VERSION}_windows_amd64
	zip -j release/tfk8s_${VERSION}_windows_amd64.zip release/tfk8s_${VERSION}_windows_amd64
	GOOS=windows GOOARCH=386 go build -ldflags "-X main.toolVersion=${VERSION}" -o release/tfk8s_${VERSION}_windows_386
	zip -j release/tfk8s_${VERSION}_windows_386.zip release/tfk8s_${VERSION}_windows_386

install: 
	go install -ldflags "-X main.toolVersion=${VERSION}"

test:
	go test -v ./...

clean:
	rm -rf release/*
