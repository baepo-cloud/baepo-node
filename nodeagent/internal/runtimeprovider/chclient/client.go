package chclient

//go:generate go tool oapi-codegen -generate client -o client.gen.go -package chclient ../../resources/cloud-hypervisor.swagger.yaml
//go:generate go tool oapi-codegen -generate models -o models.gen.go -package chclient ../../resources/cloud-hypervisor.swagger.yaml
