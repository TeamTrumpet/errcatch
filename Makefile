all:
	@echo "Compiling assets..."
	@go-bindata templates
	@echo "Compiling binary..."
	@go build -v
	@echo "Done!"
