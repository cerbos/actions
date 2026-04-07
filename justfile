default:
  @ just list

update-toolbox:
  @ go -C hack/go run ./cmd/update-toolbox
  @ cd hack/js && pnpm install && pnpm run build

wrap-actions:
  @ go -C hack/go run ./cmd/wrap-actions
