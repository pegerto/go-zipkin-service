all:
	docker run --rm -v "$(shell pwd)":/usr/src/myapp -w /usr/src/myapp tcnksm/gox:1.7  go get; gox -osarch="darwin/amd64" -output "pkg/product_service_{{.OS}}_{{.Arch}}/{{.Dir}}"
