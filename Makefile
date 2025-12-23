.PHONY: proto clean

proto:
	mkdir -p shared/generated/sensorpb shared/generated/worldpb
	protoc \
		--go_out=./shared/generated \
		--go_opt=paths=source_relative \
		--go-grpc_out=./shared/generated \
		--go-grpc_opt=paths=source_relative \
		-I./proto \
		proto/sensor.proto \
		proto/world.proto

clean:
	rm -rf shared/generated/*

