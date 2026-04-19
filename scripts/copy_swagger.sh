#!/bin/bash
# Copies all backend swagger.json files and swagger-ui static files into gateway/swagger/
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(dirname "$SCRIPT_DIR")"
GATEWAY_SWAGGER_DIR="$REPO_ROOT/gateway/swagger"

mkdir -p "$GATEWAY_SWAGGER_DIR/swagger-ui"

# Copy swagger.json files
cp "$REPO_ROOT/simplebank/doc/swagger/simple_bank.swagger.json" "$GATEWAY_SWAGGER_DIR/"
cp "$REPO_ROOT/escrow-bounty/doc/swagger/escrow_bounty.swagger.json" "$GATEWAY_SWAGGER_DIR/"
cp "$REPO_ROOT/user-profile-service/doc/swagger/user_profile.swagger.json" "$GATEWAY_SWAGGER_DIR/"
cp "$REPO_ROOT/payment-service/doc/swagger/payment_service.swagger.json" "$GATEWAY_SWAGGER_DIR/"

# Copy swagger-ui static files
cp "$REPO_ROOT/simplebank/doc/swagger/"*.css "$GATEWAY_SWAGGER_DIR/swagger-ui/"
cp "$REPO_ROOT/simplebank/doc/swagger/"*.js "$GATEWAY_SWAGGER_DIR/swagger-ui/"
cp "$REPO_ROOT/simplebank/doc/swagger/"*.html "$GATEWAY_SWAGGER_DIR/swagger-ui/"
cp "$REPO_ROOT/simplebank/doc/swagger/"*.png "$GATEWAY_SWAGGER_DIR/swagger-ui/"

echo "Swagger files copied to $GATEWAY_SWAGGER_DIR"
