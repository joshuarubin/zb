GOVENDOR ?= govendor

save:
	$(GOVENDOR) init
	$(GOVENDOR) remove +unused
	GOOS=linux GOARCH=amd64 $(GOVENDOR) update +vendor
	GOOS=linux GOARCH=amd64 $(GOVENDOR) add +external,^program
.PHONY: save
