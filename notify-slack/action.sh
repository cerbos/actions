#!/usr/bin/env bash
set -o errexit
set -o nounset
set -o pipefail

case "${JOB_STATUS}" in
  success)
    color="good"
    ;;
  failure)
    color="danger"
    ;;
  cancelled)
    color="warning"
    ;;
esac

repository_url="${GITHUB_SERVER_URL}/${GITHUB_REPOSITORY}"

if [[ "${GITHUB_EVENT_NAME}" = "pull_request" ]]; then
  ref_title="Pull request"
  ref_link="<${repository_url}/pull/${PULL_REQUEST} | #${PULL_REQUEST}>"
else
  ref_title="Commit"
  ref_link="<${repository_url}/commit/${GITHUB_SHA} | ${GITHUB_SHA::7}> (${GITHUB_REF_NAME})"
fi

repository_link="<${repository_url} | ${GITHUB_REPOSITORY}>"
workflow_link="<${repository_url}/actions/runs/${GITHUB_RUN_ID}/attempts/${GITHUB_RUN_ATTEMPT} | ${GITHUB_WORKFLOW} #${GITHUB_RUN_NUMBER}>"

message=$(
  jq \
    --arg attempt "${GITHUB_RUN_ATTEMPT}" \
    --arg color "${color}" \
    --arg event "${GITHUB_EVENT_NAME}" \
    --arg refLink "${ref_link}" \
    --arg refTitle "${ref_title}" \
    --arg repositoryLink "${repository_link}" \
    --arg status "${JOB_STATUS}" \
    --arg workflowLink "${workflow_link}" \
    --null-input \
    '{
      attachments: [{
        color: $color,
        fields: [
          {
            title: "Repository",
            value: $repositoryLink,
            short: true
          },
          {
            title: "Workflow",
            value: $workflowLink,
            short: true
          },
          {
            title: $refTitle,
            value: $refLink,
            short: true
          },
          {
            title: "Attempt",
            value: $attempt,
            short: true
          },
          {
            title: "Status",
            value: $status,
            short: true
          },
          {
            title: "Event",
            value: $event,
            short: true
          }
        ],
        footer_icon: "https://slack.github.com/static/img/favicon-neutral.png",
        footer: $repositoryLink,
        ts: (now | floor)
      }]
    }'
)

curl \
  --fail \
  --output /dev/null \
  --silent \
  --show-error \
  --json "${message}" \
  "${SLACK_WEBHOOK_URL}"
