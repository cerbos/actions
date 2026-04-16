default:
  @ just list

update-toolbox:
  @ go run ./hack/go/cmd/update-toolbox
  @ cd hack/js && pnpm install && pnpm run build

wrap-actions:
  @ go run ./hack/go/cmd/wrap-actions
