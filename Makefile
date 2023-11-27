STACK = days
VERSION = $(shell git rev-parse --abbrev-ref HEAD)-$(shell git rev-parse --short HEAD)
SAM_CLI_TELEMETRY=0


.PHONY: build deploy validate destroy

DOMAINNAME = days.dabase.com
ACMCERTIFICATEARN = arn:aws:acm:eu-west-2:407461997746:certificate/9083a66b-72b6-448d-9bce-6ee2e2e52e36


deploy:
	sam build
	SAM_CLI_TELEMETRY=0 sam deploy --resolve-s3 --stack-name $(STACK) --parameter-overrides DomainName=$(DOMAINNAME) ACMCertificateArn=$(ACMCERTIFICATEARN) --no-confirm-changeset --no-fail-on-empty-changeset --capabilities CAPABILITY_IAM --disable-rollback

build-MainFunction:
	cp LICENSE ${ARTIFACTS_DIR}
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags "-X main.Version=$(VERSION)" -o ${ARTIFACTS_DIR}/bootstrap

validate:
	aws cloudformation validate-template --template-body file://template.yml

destroy:
	aws cloudformation delete-stack --stack-name $(STACK)

sam-tail-logs:
	sam logs --stack-name $(STACK) --tail

clean:
	rm -rf main gin-bin
