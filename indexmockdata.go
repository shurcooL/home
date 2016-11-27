package main

const sampleEventsData = `[
  {
    "id": "4872223251",
    "type": "IssueCommentEvent",
    "actor": {
      "id": 1924134,
      "login": "shurcooL",
      "display_login": "shurcooL",
      "gravatar_id": "",
      "url": "https://api.github.com/users/shurcooL",
      "avatar_url": "https://avatars.githubusercontent.com/u/1924134?"
    },
    "repo": {
      "id": 62099451,
      "name": "campoy/embedmd",
      "url": "https://api.github.com/repos/campoy/embedmd"
    },
    "payload": {
      "action": "created",
      "issue": {
        "url": "https://api.github.com/repos/campoy/embedmd/issues/28",
        "repository_url": "https://api.github.com/repos/campoy/embedmd",
        "labels_url": "https://api.github.com/repos/campoy/embedmd/issues/28/labels{/name}",
        "comments_url": "https://api.github.com/repos/campoy/embedmd/issues/28/comments",
        "events_url": "https://api.github.com/repos/campoy/embedmd/issues/28/events",
        "html_url": "https://github.com/campoy/embedmd/pull/28",
        "id": 188954562,
        "number": 28,
        "title": "extracting the main functionality into a resuable lib",
        "user": {
          "login": "campoy",
          "id": 2237452,
          "avatar_url": "https://avatars.githubusercontent.com/u/2237452?v=3",
          "gravatar_id": "",
          "url": "https://api.github.com/users/campoy",
          "html_url": "https://github.com/campoy",
          "followers_url": "https://api.github.com/users/campoy/followers",
          "following_url": "https://api.github.com/users/campoy/following{/other_user}",
          "gists_url": "https://api.github.com/users/campoy/gists{/gist_id}",
          "starred_url": "https://api.github.com/users/campoy/starred{/owner}{/repo}",
          "subscriptions_url": "https://api.github.com/users/campoy/subscriptions",
          "organizations_url": "https://api.github.com/users/campoy/orgs",
          "repos_url": "https://api.github.com/users/campoy/repos",
          "events_url": "https://api.github.com/users/campoy/events{/privacy}",
          "received_events_url": "https://api.github.com/users/campoy/received_events",
          "type": "User",
          "site_admin": false
        },
        "labels": [

        ],
        "state": "open",
        "locked": false,
        "assignee": null,
        "assignees": [

        ],
        "milestone": null,
        "comments": 5,
        "created_at": "2016-11-13T06:15:16Z",
        "updated_at": "2016-11-15T20:18:35Z",
        "closed_at": null,
        "pull_request": {
          "url": "https://api.github.com/repos/campoy/embedmd/pulls/28",
          "html_url": "https://github.com/campoy/embedmd/pull/28",
          "diff_url": "https://github.com/campoy/embedmd/pull/28.diff",
          "patch_url": "https://github.com/campoy/embedmd/pull/28.patch"
        },
        "body": "Fixes #10."
      },
      "comment": {
        "url": "https://api.github.com/repos/campoy/embedmd/issues/comments/260755327",
        "html_url": "https://github.com/campoy/embedmd/pull/28#issuecomment-260755327",
        "issue_url": "https://api.github.com/repos/campoy/embedmd/issues/28",
        "id": 260755327,
        "user": {
          "login": "shurcooL",
          "id": 1924134,
          "avatar_url": "https://avatars.githubusercontent.com/u/1924134?v=3",
          "gravatar_id": "",
          "url": "https://api.github.com/users/shurcooL",
          "html_url": "https://github.com/shurcooL",
          "followers_url": "https://api.github.com/users/shurcooL/followers",
          "following_url": "https://api.github.com/users/shurcooL/following{/other_user}",
          "gists_url": "https://api.github.com/users/shurcooL/gists{/gist_id}",
          "starred_url": "https://api.github.com/users/shurcooL/starred{/owner}{/repo}",
          "subscriptions_url": "https://api.github.com/users/shurcooL/subscriptions",
          "organizations_url": "https://api.github.com/users/shurcooL/orgs",
          "repos_url": "https://api.github.com/users/shurcooL/repos",
          "events_url": "https://api.github.com/users/shurcooL/events{/privacy}",
          "received_events_url": "https://api.github.com/users/shurcooL/received_events",
          "type": "User",
          "site_admin": false
        },
        "created_at": "2016-11-15T20:18:35Z",
        "updated_at": "2016-11-15T20:18:35Z",
        "body": "A high level comment for you to consider (but not necessarily do) is to keep the library at ` + "`" + `github.com/campoy/embedmd` + "`" + ` import path, and move the command to ` + "`" + `github.com/campoy/embedmd/cmd/embedmd` + "`" + `.\r\n\r\nThe ` + "`" + `cmd/foo` + "`" + ` pattern is pretty common and I think it's nice. It lets you avoid the stuttering in ` + "`" + `github.com/campoy/embedmd/embedmd` + "`" + ` import path, and makes it more visible where the command/libraries are. But it's definitely a matter of taste, so completely up to you.\r\n\r\nSome examples of the pattern:\r\n\r\n- https://godoc.org/golang.org/x/tools/cmd\r\n- https://github.com/bradfitz/issuemirror/commit/af29bba36e54185782b614beeacd310a01a62211\r\n- https://github.com/dominikh/go-simple/tree/master/cmd/gosimple"
      }
    },
    "public": true,
    "created_at": "2016-11-15T20:18:35Z"
  },
  {
    "id": "4872117478",
    "type": "PushEvent",
    "actor": {
      "id": 1924134,
      "login": "shurcooL",
      "display_login": "shurcooL",
      "gravatar_id": "",
      "url": "https://api.github.com/users/shurcooL",
      "avatar_url": "https://avatars.githubusercontent.com/u/1924134?"
    },
    "repo": {
      "id": 73355224,
      "name": "shurcooL/SLA",
      "url": "https://api.github.com/repos/shurcooL/SLA"
    },
    "payload": {
      "push_id": 1404528266,
      "size": 1,
      "distinct_size": 1,
      "ref": "refs/heads/master",
      "head": "7a2be3ee4a85aa385362635251a5876240e4c2f2",
      "before": "e869b9c7dc9a3795ae0e6cec281b9d3dcb94ce08",
      "commits": [
        {
          "sha": "7a2be3ee4a85aa385362635251a5876240e4c2f2",
          "author": {
            "email": "shurcooL@gmail.com",
            "name": "Dmitri Shuralyov"
          },
          "message": "Cover goxjs packages in applicability.\n\nThey are in scope of the SLA as well.",
          "distinct": true,
          "url": "https://api.github.com/repos/shurcooL/SLA/commits/7a2be3ee4a85aa385362635251a5876240e4c2f2"
        }
      ]
    },
    "public": true,
    "created_at": "2016-11-15T20:00:10Z"
  },
  {
    "id": "4871998823",
    "type": "IssueCommentEvent",
    "actor": {
      "id": 1924134,
      "login": "shurcooL",
      "display_login": "shurcooL",
      "gravatar_id": "",
      "url": "https://api.github.com/users/shurcooL",
      "avatar_url": "https://avatars.githubusercontent.com/u/1924134?"
    },
    "repo": {
      "id": 10151943,
      "name": "go-gl/glfw",
      "url": "https://api.github.com/repos/go-gl/glfw"
    },
    "payload": {
      "action": "created",
      "issue": {
        "url": "https://api.github.com/repos/go-gl/glfw/issues/177",
        "repository_url": "https://api.github.com/repos/go-gl/glfw",
        "labels_url": "https://api.github.com/repos/go-gl/glfw/issues/177/labels{/name}",
        "comments_url": "https://api.github.com/repos/go-gl/glfw/issues/177/comments",
        "events_url": "https://api.github.com/repos/go-gl/glfw/issues/177/events",
        "html_url": "https://github.com/go-gl/glfw/issues/177",
        "id": 189155147,
        "number": 177,
        "title": "VS CODE + DELVE + GLFW V3.2 PROBLEM",
        "user": {
          "login": "MrLiet",
          "id": 23429917,
          "avatar_url": "https://avatars.githubusercontent.com/u/23429917?v=3",
          "gravatar_id": "",
          "url": "https://api.github.com/users/MrLiet",
          "html_url": "https://github.com/MrLiet",
          "followers_url": "https://api.github.com/users/MrLiet/followers",
          "following_url": "https://api.github.com/users/MrLiet/following{/other_user}",
          "gists_url": "https://api.github.com/users/MrLiet/gists{/gist_id}",
          "starred_url": "https://api.github.com/users/MrLiet/starred{/owner}{/repo}",
          "subscriptions_url": "https://api.github.com/users/MrLiet/subscriptions",
          "organizations_url": "https://api.github.com/users/MrLiet/orgs",
          "repos_url": "https://api.github.com/users/MrLiet/repos",
          "events_url": "https://api.github.com/users/MrLiet/events{/privacy}",
          "received_events_url": "https://api.github.com/users/MrLiet/received_events",
          "type": "User",
          "site_admin": false
        },
        "labels": [

        ],
        "state": "open",
        "locked": false,
        "assignee": null,
        "assignees": [

        ],
        "milestone": null,
        "comments": 1,
        "created_at": "2016-11-14T16:17:28Z",
        "updated_at": "2016-11-15T19:40:35Z",
        "closed_at": null,
        "body": "Hello.\r\n\r\nI have compile error:\r\n\r\n![image](https://cloud.githubusercontent.com/assets/23429917/20272702/4ff18ac6-aac1-11e6-9e7b-215ac472d93c.png)\r\n\r\n\r\nSource code:\r\n\r\n` + "```" + `\r\npackage main\r\n\r\nimport \"github.com/go-gl/glfw/v3.2/glfw\"\r\nimport \"runtime\"\r\n\r\nfunc main() {\r\n\r\n\truntime.LockOSThread()\r\n\r\n\tglfw.Init()\r\n\r\n\tw, _ := glfw.CreateWindow(800, 600, \"hello\", nil, nil)\r\n\r\n\tfor {\r\n\t\tglfw.PollEvents()\r\n\t\tif w.ShouldClose() {\r\n\t\t\tbreak\r\n\t\t}\r\n\t}\r\n\r\n}\r\n` + "```" + `\r\n\r\nConsole output:\r\n\r\n> me/proj1\r\n> github.com/go-gl/glfw/v3.2/glfw(.text): strdup: not defined\r\n> github.com/go-gl/glfw/v3.2/glfw(.text): strdup: not defined\r\n> github.com/go-gl/glfw/v3.2/glfw(.text): strdup: not defined\r\n> github.com/go-gl/glfw/v3.2/glfw(.text): strdup: not defined\r\n> github.com/go-gl/glfw/v3.2/glfw(.text): undefined: strdup\r\n> github.com/go-gl/glfw/v3.2/glfw(.text): undefined: strdup\r\n> github.com/go-gl/glfw/v3.2/glfw(.text): undefined: strdup\r\n> github.com/go-gl/glfw/v3.2/glfw(.text): undefined: strdup\r\n> exit status 2\r\n> \r\n\r\nIm use:\r\n\r\n- Windows 64 bit\r\n- Go 1.7.3 \r\n- Mingw 64\r\n- VS Code\r\n\r\nVS Code settings:\r\n` + "```" + `\r\n{\r\n    \"go.buildOnSave\": true,\r\n    \"go.lintOnSave\": true,\r\n    \"go.vetOnSave\": true,\r\n    \"go.buildTags\": \"\",\r\n    \"go.buildFlags\": [],\r\n    \"go.lintTool\": \"golint\",\r\n    \"go.lintFlags\": [],\r\n    \"go.vetFlags\": [],\r\n    \"go.coverOnSave\":false,\r\n    \"go.useCodeSnippetsOnFunctionSuggest\": true,\r\n    \"go.formatOnSave\": true, \r\n    \"go.formatTool\": \"goreturns\",\r\n    \"go.formatFlags\": [],\r\n    \"go.gocodeAutoBuild\": false,\r\n    \"go.autocompleteUnimportedPackages\": true\r\n}\r\n` + "```" + `\r\n\r\nand Launch settings:\r\n` + "```" + `\r\n{\r\n    \"version\": \"0.2.0\",\r\n    \"configurations\": [\r\n        {\r\n            \"name\": \"Launch\",\r\n            \"type\": \"go\",\r\n            \"request\": \"launch\",\r\n            \"mode\": \"debug\",\r\n            \"program\": \"${workspaceRoot}\",\r\n            \"env\": {},\r\n            \"args\": []\r\n        }\r\n    ]\r\n}\r\n` + "```" + `\r\n\r\nAnd it's work fine with glfw 3.1 version\r\n![image](https://cloud.githubusercontent.com/assets/23429917/20272758/7aef4be6-aac1-11e6-8d41-a8a5081817a6.png)\r\n\r\n\r\nHelp!"
      },
      "comment": {
        "url": "https://api.github.com/repos/go-gl/glfw/issues/comments/260744573",
        "html_url": "https://github.com/go-gl/glfw/issues/177#issuecomment-260744573",
        "issue_url": "https://api.github.com/repos/go-gl/glfw/issues/177",
        "id": 260744573,
        "user": {
          "login": "shurcooL",
          "id": 1924134,
          "avatar_url": "https://avatars.githubusercontent.com/u/1924134?v=3",
          "gravatar_id": "",
          "url": "https://api.github.com/users/shurcooL",
          "html_url": "https://github.com/shurcooL",
          "followers_url": "https://api.github.com/users/shurcooL/followers",
          "following_url": "https://api.github.com/users/shurcooL/following{/other_user}",
          "gists_url": "https://api.github.com/users/shurcooL/gists{/gist_id}",
          "starred_url": "https://api.github.com/users/shurcooL/starred{/owner}{/repo}",
          "subscriptions_url": "https://api.github.com/users/shurcooL/subscriptions",
          "organizations_url": "https://api.github.com/users/shurcooL/orgs",
          "repos_url": "https://api.github.com/users/shurcooL/repos",
          "events_url": "https://api.github.com/users/shurcooL/events{/privacy}",
          "received_events_url": "https://api.github.com/users/shurcooL/received_events",
          "type": "User",
          "site_admin": false
        },
        "created_at": "2016-11-15T19:40:35Z",
        "updated_at": "2016-11-15T19:40:35Z",
        "body": "@tapir, he said he's using Go 1.7.3, so it's not an old version of Go.\r\n\r\n@MrLiet, can you run the following 4 commands and share their output exactly:\r\n\r\n` + "```" + `\r\ngo version\r\ngo env\r\ngo get -u -v github.com/go-gl/glfw/v3.1/glfw\r\ngo get -u -v github.com/go-gl/glfw/v3.2/glfw\r\n` + "```" + `\r\n\r\nIf that prints the same error about ` + "`" + `strdup` + "`" + ` being undefined for 3.2 only, it might be that we messed up some import in the headers on Windows. Can anyone else confirm if 3.2 works on Windows for them?"
      }
    },
    "public": true,
    "created_at": "2016-11-15T19:40:35Z",
    "org": {
      "id": 2505184,
      "login": "go-gl",
      "gravatar_id": "",
      "url": "https://api.github.com/orgs/go-gl",
      "avatar_url": "https://avatars.githubusercontent.com/u/2505184?"
    }
  },
  {
    "id": "4871916854",
    "type": "PullRequestReviewCommentEvent",
    "actor": {
      "id": 1924134,
      "login": "shurcooL",
      "display_login": "shurcooL",
      "gravatar_id": "",
      "url": "https://api.github.com/users/shurcooL",
      "avatar_url": "https://avatars.githubusercontent.com/u/1924134?"
    },
    "repo": {
      "id": 62099451,
      "name": "campoy/embedmd",
      "url": "https://api.github.com/repos/campoy/embedmd"
    },
    "payload": {
      "action": "created",
      "comment": {
        "url": "https://api.github.com/repos/campoy/embedmd/pulls/comments/88093459",
        "pull_request_review_id": 8677336,
        "id": 88093459,
        "diff_hunk": "@@ -0,0 +1,48 @@\n+// Copyright 2016 Google Inc. All rights reserved.\n+// Licensed under the Apache License, Version 2.0 (the \"License\");\n+// you may not use this file except in compliance with the License.\n+// You may obtain a copy of the License at\n+// http://www.apache.org/licenses/LICENSE-2.0\n+//\n+// Unless required by applicable law or agreed to writing, software distributed\n+// under the License is distributed on a \"AS IS\" BASIS, WITHOUT WARRANTIES OR\n+// CONDITIONS OF ANY KIND, either express or implied.\n+//\n+// See the License for the specific language governing permissions and\n+// limitations under the License.\n+\n+package embedmd\n+\n+import (\n+\t\"fmt\"\n+\t\"io/ioutil\"\n+\t\"net/http\"\n+\t\"path/filepath\"\n+\t\"strings\"\n+)\n+\n+// Fetcher provides an abstraction on a file system.\n+// The Fetch function is called anytime some content needs to be fetched.\n+// For now this includes files and URLs.\n+type Fetcher interface {\n+\tFetch(dir, path string) ([]byte, error)",
        "path": "embedmd/content.go",
        "position": null,
        "original_position": 28,
        "commit_id": "9cab4535ee4bc92a916f6cc5e300060b61d6439b",
        "original_commit_id": "7e3080602f24bb83fcd0ce194538f1e1cedb7cfc",
        "user": {
          "login": "shurcooL",
          "id": 1924134,
          "avatar_url": "https://avatars.githubusercontent.com/u/1924134?v=3",
          "gravatar_id": "",
          "url": "https://api.github.com/users/shurcooL",
          "html_url": "https://github.com/shurcooL",
          "followers_url": "https://api.github.com/users/shurcooL/followers",
          "following_url": "https://api.github.com/users/shurcooL/following{/other_user}",
          "gists_url": "https://api.github.com/users/shurcooL/gists{/gist_id}",
          "starred_url": "https://api.github.com/users/shurcooL/starred{/owner}{/repo}",
          "subscriptions_url": "https://api.github.com/users/shurcooL/subscriptions",
          "organizations_url": "https://api.github.com/users/shurcooL/orgs",
          "repos_url": "https://api.github.com/users/shurcooL/repos",
          "events_url": "https://api.github.com/users/shurcooL/events{/privacy}",
          "received_events_url": "https://api.github.com/users/shurcooL/received_events",
          "type": "User",
          "site_admin": false
        },
        "body": "> I added some docs.\r\n\r\nThank you, those docs help a _ton_.",
        "created_at": "2016-11-15T19:26:46Z",
        "updated_at": "2016-11-15T19:26:46Z",
        "html_url": "https://github.com/campoy/embedmd/pull/28#discussion_r88093459",
        "pull_request_url": "https://api.github.com/repos/campoy/embedmd/pulls/28",
        "_links": {
          "self": {
            "href": "https://api.github.com/repos/campoy/embedmd/pulls/comments/88093459"
          },
          "html": {
            "href": "https://github.com/campoy/embedmd/pull/28#discussion_r88093459"
          },
          "pull_request": {
            "href": "https://api.github.com/repos/campoy/embedmd/pulls/28"
          }
        }
      },
      "pull_request": {
        "url": "https://api.github.com/repos/campoy/embedmd/pulls/28",
        "id": 93463783,
        "html_url": "https://github.com/campoy/embedmd/pull/28",
        "diff_url": "https://github.com/campoy/embedmd/pull/28.diff",
        "patch_url": "https://github.com/campoy/embedmd/pull/28.patch",
        "issue_url": "https://api.github.com/repos/campoy/embedmd/issues/28",
        "number": 28,
        "state": "open",
        "locked": false,
        "title": "extracting the main functionality into a resuable lib",
        "user": {
          "login": "campoy",
          "id": 2237452,
          "avatar_url": "https://avatars.githubusercontent.com/u/2237452?v=3",
          "gravatar_id": "",
          "url": "https://api.github.com/users/campoy",
          "html_url": "https://github.com/campoy",
          "followers_url": "https://api.github.com/users/campoy/followers",
          "following_url": "https://api.github.com/users/campoy/following{/other_user}",
          "gists_url": "https://api.github.com/users/campoy/gists{/gist_id}",
          "starred_url": "https://api.github.com/users/campoy/starred{/owner}{/repo}",
          "subscriptions_url": "https://api.github.com/users/campoy/subscriptions",
          "organizations_url": "https://api.github.com/users/campoy/orgs",
          "repos_url": "https://api.github.com/users/campoy/repos",
          "events_url": "https://api.github.com/users/campoy/events{/privacy}",
          "received_events_url": "https://api.github.com/users/campoy/received_events",
          "type": "User",
          "site_admin": false
        },
        "body": "Fixes #10.",
        "created_at": "2016-11-13T06:15:16Z",
        "updated_at": "2016-11-15T19:26:46Z",
        "closed_at": null,
        "merged_at": null,
        "merge_commit_sha": "85ffa9c2497857fc9d3b8e21d775a443c9247588",
        "assignee": null,
        "assignees": [

        ],
        "milestone": null,
        "commits_url": "https://api.github.com/repos/campoy/embedmd/pulls/28/commits",
        "review_comments_url": "https://api.github.com/repos/campoy/embedmd/pulls/28/comments",
        "review_comment_url": "https://api.github.com/repos/campoy/embedmd/pulls/comments{/number}",
        "comments_url": "https://api.github.com/repos/campoy/embedmd/issues/28/comments",
        "statuses_url": "https://api.github.com/repos/campoy/embedmd/statuses/9cab4535ee4bc92a916f6cc5e300060b61d6439b",
        "head": {
          "label": "campoy:lib",
          "ref": "lib",
          "sha": "9cab4535ee4bc92a916f6cc5e300060b61d6439b",
          "user": {
            "login": "campoy",
            "id": 2237452,
            "avatar_url": "https://avatars.githubusercontent.com/u/2237452?v=3",
            "gravatar_id": "",
            "url": "https://api.github.com/users/campoy",
            "html_url": "https://github.com/campoy",
            "followers_url": "https://api.github.com/users/campoy/followers",
            "following_url": "https://api.github.com/users/campoy/following{/other_user}",
            "gists_url": "https://api.github.com/users/campoy/gists{/gist_id}",
            "starred_url": "https://api.github.com/users/campoy/starred{/owner}{/repo}",
            "subscriptions_url": "https://api.github.com/users/campoy/subscriptions",
            "organizations_url": "https://api.github.com/users/campoy/orgs",
            "repos_url": "https://api.github.com/users/campoy/repos",
            "events_url": "https://api.github.com/users/campoy/events{/privacy}",
            "received_events_url": "https://api.github.com/users/campoy/received_events",
            "type": "User",
            "site_admin": false
          },
          "repo": {
            "id": 62099451,
            "name": "embedmd",
            "full_name": "campoy/embedmd",
            "owner": {
              "login": "campoy",
              "id": 2237452,
              "avatar_url": "https://avatars.githubusercontent.com/u/2237452?v=3",
              "gravatar_id": "",
              "url": "https://api.github.com/users/campoy",
              "html_url": "https://github.com/campoy",
              "followers_url": "https://api.github.com/users/campoy/followers",
              "following_url": "https://api.github.com/users/campoy/following{/other_user}",
              "gists_url": "https://api.github.com/users/campoy/gists{/gist_id}",
              "starred_url": "https://api.github.com/users/campoy/starred{/owner}{/repo}",
              "subscriptions_url": "https://api.github.com/users/campoy/subscriptions",
              "organizations_url": "https://api.github.com/users/campoy/orgs",
              "repos_url": "https://api.github.com/users/campoy/repos",
              "events_url": "https://api.github.com/users/campoy/events{/privacy}",
              "received_events_url": "https://api.github.com/users/campoy/received_events",
              "type": "User",
              "site_admin": false
            },
            "private": false,
            "html_url": "https://github.com/campoy/embedmd",
            "description": "embedmd: embed code into markdown and keep everything in sync",
            "fork": false,
            "url": "https://api.github.com/repos/campoy/embedmd",
            "forks_url": "https://api.github.com/repos/campoy/embedmd/forks",
            "keys_url": "https://api.github.com/repos/campoy/embedmd/keys{/key_id}",
            "collaborators_url": "https://api.github.com/repos/campoy/embedmd/collaborators{/collaborator}",
            "teams_url": "https://api.github.com/repos/campoy/embedmd/teams",
            "hooks_url": "https://api.github.com/repos/campoy/embedmd/hooks",
            "issue_events_url": "https://api.github.com/repos/campoy/embedmd/issues/events{/number}",
            "events_url": "https://api.github.com/repos/campoy/embedmd/events",
            "assignees_url": "https://api.github.com/repos/campoy/embedmd/assignees{/user}",
            "branches_url": "https://api.github.com/repos/campoy/embedmd/branches{/branch}",
            "tags_url": "https://api.github.com/repos/campoy/embedmd/tags",
            "blobs_url": "https://api.github.com/repos/campoy/embedmd/git/blobs{/sha}",
            "git_tags_url": "https://api.github.com/repos/campoy/embedmd/git/tags{/sha}",
            "git_refs_url": "https://api.github.com/repos/campoy/embedmd/git/refs{/sha}",
            "trees_url": "https://api.github.com/repos/campoy/embedmd/git/trees{/sha}",
            "statuses_url": "https://api.github.com/repos/campoy/embedmd/statuses/{sha}",
            "languages_url": "https://api.github.com/repos/campoy/embedmd/languages",
            "stargazers_url": "https://api.github.com/repos/campoy/embedmd/stargazers",
            "contributors_url": "https://api.github.com/repos/campoy/embedmd/contributors",
            "subscribers_url": "https://api.github.com/repos/campoy/embedmd/subscribers",
            "subscription_url": "https://api.github.com/repos/campoy/embedmd/subscription",
            "commits_url": "https://api.github.com/repos/campoy/embedmd/commits{/sha}",
            "git_commits_url": "https://api.github.com/repos/campoy/embedmd/git/commits{/sha}",
            "comments_url": "https://api.github.com/repos/campoy/embedmd/comments{/number}",
            "issue_comment_url": "https://api.github.com/repos/campoy/embedmd/issues/comments{/number}",
            "contents_url": "https://api.github.com/repos/campoy/embedmd/contents/{+path}",
            "compare_url": "https://api.github.com/repos/campoy/embedmd/compare/{base}...{head}",
            "merges_url": "https://api.github.com/repos/campoy/embedmd/merges",
            "archive_url": "https://api.github.com/repos/campoy/embedmd/{archive_format}{/ref}",
            "downloads_url": "https://api.github.com/repos/campoy/embedmd/downloads",
            "issues_url": "https://api.github.com/repos/campoy/embedmd/issues{/number}",
            "pulls_url": "https://api.github.com/repos/campoy/embedmd/pulls{/number}",
            "milestones_url": "https://api.github.com/repos/campoy/embedmd/milestones{/number}",
            "notifications_url": "https://api.github.com/repos/campoy/embedmd/notifications{?since,all,participating}",
            "labels_url": "https://api.github.com/repos/campoy/embedmd/labels{/name}",
            "releases_url": "https://api.github.com/repos/campoy/embedmd/releases{/id}",
            "deployments_url": "https://api.github.com/repos/campoy/embedmd/deployments",
            "created_at": "2016-06-28T01:16:46Z",
            "updated_at": "2016-11-15T17:37:47Z",
            "pushed_at": "2016-11-15T19:19:42Z",
            "git_url": "git://github.com/campoy/embedmd.git",
            "ssh_url": "git@github.com:campoy/embedmd.git",
            "clone_url": "https://github.com/campoy/embedmd.git",
            "svn_url": "https://github.com/campoy/embedmd",
            "homepage": "",
            "size": 76,
            "stargazers_count": 297,
            "watchers_count": 297,
            "language": "Go",
            "has_issues": true,
            "has_downloads": true,
            "has_wiki": true,
            "has_pages": false,
            "forks_count": 9,
            "mirror_url": null,
            "open_issues_count": 3,
            "forks": 9,
            "open_issues": 3,
            "watchers": 297,
            "default_branch": "master"
          }
        },
        "base": {
          "label": "campoy:master",
          "ref": "master",
          "sha": "c005cb67a74ca57cf27d2db719c3e244cb440548",
          "user": {
            "login": "campoy",
            "id": 2237452,
            "avatar_url": "https://avatars.githubusercontent.com/u/2237452?v=3",
            "gravatar_id": "",
            "url": "https://api.github.com/users/campoy",
            "html_url": "https://github.com/campoy",
            "followers_url": "https://api.github.com/users/campoy/followers",
            "following_url": "https://api.github.com/users/campoy/following{/other_user}",
            "gists_url": "https://api.github.com/users/campoy/gists{/gist_id}",
            "starred_url": "https://api.github.com/users/campoy/starred{/owner}{/repo}",
            "subscriptions_url": "https://api.github.com/users/campoy/subscriptions",
            "organizations_url": "https://api.github.com/users/campoy/orgs",
            "repos_url": "https://api.github.com/users/campoy/repos",
            "events_url": "https://api.github.com/users/campoy/events{/privacy}",
            "received_events_url": "https://api.github.com/users/campoy/received_events",
            "type": "User",
            "site_admin": false
          },
          "repo": {
            "id": 62099451,
            "name": "embedmd",
            "full_name": "campoy/embedmd",
            "owner": {
              "login": "campoy",
              "id": 2237452,
              "avatar_url": "https://avatars.githubusercontent.com/u/2237452?v=3",
              "gravatar_id": "",
              "url": "https://api.github.com/users/campoy",
              "html_url": "https://github.com/campoy",
              "followers_url": "https://api.github.com/users/campoy/followers",
              "following_url": "https://api.github.com/users/campoy/following{/other_user}",
              "gists_url": "https://api.github.com/users/campoy/gists{/gist_id}",
              "starred_url": "https://api.github.com/users/campoy/starred{/owner}{/repo}",
              "subscriptions_url": "https://api.github.com/users/campoy/subscriptions",
              "organizations_url": "https://api.github.com/users/campoy/orgs",
              "repos_url": "https://api.github.com/users/campoy/repos",
              "events_url": "https://api.github.com/users/campoy/events{/privacy}",
              "received_events_url": "https://api.github.com/users/campoy/received_events",
              "type": "User",
              "site_admin": false
            },
            "private": false,
            "html_url": "https://github.com/campoy/embedmd",
            "description": "embedmd: embed code into markdown and keep everything in sync",
            "fork": false,
            "url": "https://api.github.com/repos/campoy/embedmd",
            "forks_url": "https://api.github.com/repos/campoy/embedmd/forks",
            "keys_url": "https://api.github.com/repos/campoy/embedmd/keys{/key_id}",
            "collaborators_url": "https://api.github.com/repos/campoy/embedmd/collaborators{/collaborator}",
            "teams_url": "https://api.github.com/repos/campoy/embedmd/teams",
            "hooks_url": "https://api.github.com/repos/campoy/embedmd/hooks",
            "issue_events_url": "https://api.github.com/repos/campoy/embedmd/issues/events{/number}",
            "events_url": "https://api.github.com/repos/campoy/embedmd/events",
            "assignees_url": "https://api.github.com/repos/campoy/embedmd/assignees{/user}",
            "branches_url": "https://api.github.com/repos/campoy/embedmd/branches{/branch}",
            "tags_url": "https://api.github.com/repos/campoy/embedmd/tags",
            "blobs_url": "https://api.github.com/repos/campoy/embedmd/git/blobs{/sha}",
            "git_tags_url": "https://api.github.com/repos/campoy/embedmd/git/tags{/sha}",
            "git_refs_url": "https://api.github.com/repos/campoy/embedmd/git/refs{/sha}",
            "trees_url": "https://api.github.com/repos/campoy/embedmd/git/trees{/sha}",
            "statuses_url": "https://api.github.com/repos/campoy/embedmd/statuses/{sha}",
            "languages_url": "https://api.github.com/repos/campoy/embedmd/languages",
            "stargazers_url": "https://api.github.com/repos/campoy/embedmd/stargazers",
            "contributors_url": "https://api.github.com/repos/campoy/embedmd/contributors",
            "subscribers_url": "https://api.github.com/repos/campoy/embedmd/subscribers",
            "subscription_url": "https://api.github.com/repos/campoy/embedmd/subscription",
            "commits_url": "https://api.github.com/repos/campoy/embedmd/commits{/sha}",
            "git_commits_url": "https://api.github.com/repos/campoy/embedmd/git/commits{/sha}",
            "comments_url": "https://api.github.com/repos/campoy/embedmd/comments{/number}",
            "issue_comment_url": "https://api.github.com/repos/campoy/embedmd/issues/comments{/number}",
            "contents_url": "https://api.github.com/repos/campoy/embedmd/contents/{+path}",
            "compare_url": "https://api.github.com/repos/campoy/embedmd/compare/{base}...{head}",
            "merges_url": "https://api.github.com/repos/campoy/embedmd/merges",
            "archive_url": "https://api.github.com/repos/campoy/embedmd/{archive_format}{/ref}",
            "downloads_url": "https://api.github.com/repos/campoy/embedmd/downloads",
            "issues_url": "https://api.github.com/repos/campoy/embedmd/issues{/number}",
            "pulls_url": "https://api.github.com/repos/campoy/embedmd/pulls{/number}",
            "milestones_url": "https://api.github.com/repos/campoy/embedmd/milestones{/number}",
            "notifications_url": "https://api.github.com/repos/campoy/embedmd/notifications{?since,all,participating}",
            "labels_url": "https://api.github.com/repos/campoy/embedmd/labels{/name}",
            "releases_url": "https://api.github.com/repos/campoy/embedmd/releases{/id}",
            "deployments_url": "https://api.github.com/repos/campoy/embedmd/deployments",
            "created_at": "2016-06-28T01:16:46Z",
            "updated_at": "2016-11-15T17:37:47Z",
            "pushed_at": "2016-11-15T19:19:42Z",
            "git_url": "git://github.com/campoy/embedmd.git",
            "ssh_url": "git@github.com:campoy/embedmd.git",
            "clone_url": "https://github.com/campoy/embedmd.git",
            "svn_url": "https://github.com/campoy/embedmd",
            "homepage": "",
            "size": 76,
            "stargazers_count": 297,
            "watchers_count": 297,
            "language": "Go",
            "has_issues": true,
            "has_downloads": true,
            "has_wiki": true,
            "has_pages": false,
            "forks_count": 9,
            "mirror_url": null,
            "open_issues_count": 3,
            "forks": 9,
            "open_issues": 3,
            "watchers": 297,
            "default_branch": "master"
          }
        },
        "_links": {
          "self": {
            "href": "https://api.github.com/repos/campoy/embedmd/pulls/28"
          },
          "html": {
            "href": "https://github.com/campoy/embedmd/pull/28"
          },
          "issue": {
            "href": "https://api.github.com/repos/campoy/embedmd/issues/28"
          },
          "comments": {
            "href": "https://api.github.com/repos/campoy/embedmd/issues/28/comments"
          },
          "review_comments": {
            "href": "https://api.github.com/repos/campoy/embedmd/pulls/28/comments"
          },
          "review_comment": {
            "href": "https://api.github.com/repos/campoy/embedmd/pulls/comments{/number}"
          },
          "commits": {
            "href": "https://api.github.com/repos/campoy/embedmd/pulls/28/commits"
          },
          "statuses": {
            "href": "https://api.github.com/repos/campoy/embedmd/statuses/9cab4535ee4bc92a916f6cc5e300060b61d6439b"
          }
        }
      }
    },
    "public": true,
    "created_at": "2016-11-15T19:26:46Z"
  },
  {
    "id": "4871845750",
    "type": "PushEvent",
    "actor": {
      "id": 1924134,
      "login": "shurcooL",
      "display_login": "shurcooL",
      "gravatar_id": "",
      "url": "https://api.github.com/users/shurcooL",
      "avatar_url": "https://avatars.githubusercontent.com/u/1924134?"
    },
    "repo": {
      "id": 15789675,
      "name": "shurcooL/play",
      "url": "https://api.github.com/repos/shurcooL/play"
    },
    "payload": {
      "push_id": 1404442330,
      "size": 2,
      "distinct_size": 2,
      "ref": "refs/heads/master",
      "head": "b932f815f364fe1634ef73fcda5d06bca8be0847",
      "before": "0a5d397fbe3acb2e4cb4139a24e76dc1653111f4",
      "commits": [
        {
          "sha": "2c474fea40a00b016ef2714eb4f20a056d77628d",
          "author": {
            "email": "shurcooL@gmail.com",
            "name": "Dmitri Shuralyov"
          },
          "message": "Add reproduce code for gopherjs issue 546.",
          "distinct": true,
          "url": "https://api.github.com/repos/shurcooL/play/commits/2c474fea40a00b016ef2714eb4f20a056d77628d"
        },
        {
          "sha": "b932f815f364fe1634ef73fcda5d06bca8be0847",
          "author": {
            "email": "shurcooL@gmail.com",
            "name": "Dmitri Shuralyov"
          },
          "message": "Add test program for Go CL 33158.",
          "distinct": true,
          "url": "https://api.github.com/repos/shurcooL/play/commits/b932f815f364fe1634ef73fcda5d06bca8be0847"
        }
      ]
    },
    "public": true,
    "created_at": "2016-11-15T19:14:49Z"
  },
  {
    "id": "4871417915",
    "type": "IssueCommentEvent",
    "actor": {
      "id": 1924134,
      "login": "shurcooL",
      "display_login": "shurcooL",
      "gravatar_id": "",
      "url": "https://api.github.com/users/shurcooL",
      "avatar_url": "https://avatars.githubusercontent.com/u/1924134?"
    },
    "repo": {
      "id": 26097793,
      "name": "gopherjs/vecty",
      "url": "https://api.github.com/repos/gopherjs/vecty"
    },
    "payload": {
      "action": "created",
      "issue": {
        "url": "https://api.github.com/repos/gopherjs/vecty/issues/71",
        "repository_url": "https://api.github.com/repos/gopherjs/vecty",
        "labels_url": "https://api.github.com/repos/gopherjs/vecty/issues/71/labels{/name}",
        "comments_url": "https://api.github.com/repos/gopherjs/vecty/issues/71/comments",
        "events_url": "https://api.github.com/repos/gopherjs/vecty/issues/71/events",
        "html_url": "https://github.com/gopherjs/vecty/pull/71",
        "id": 189458236,
        "number": 71,
        "title": "Prevent reuse of range variable in closure",
        "user": {
          "login": "davelondon",
          "id": 925351,
          "avatar_url": "https://avatars.githubusercontent.com/u/925351?v=3",
          "gravatar_id": "",
          "url": "https://api.github.com/users/davelondon",
          "html_url": "https://github.com/davelondon",
          "followers_url": "https://api.github.com/users/davelondon/followers",
          "following_url": "https://api.github.com/users/davelondon/following{/other_user}",
          "gists_url": "https://api.github.com/users/davelondon/gists{/gist_id}",
          "starred_url": "https://api.github.com/users/davelondon/starred{/owner}{/repo}",
          "subscriptions_url": "https://api.github.com/users/davelondon/subscriptions",
          "organizations_url": "https://api.github.com/users/davelondon/orgs",
          "repos_url": "https://api.github.com/users/davelondon/repos",
          "events_url": "https://api.github.com/users/davelondon/events{/privacy}",
          "received_events_url": "https://api.github.com/users/davelondon/received_events",
          "type": "User",
          "site_admin": false
        },
        "labels": [

        ],
        "state": "open",
        "locked": false,
        "assignee": null,
        "assignees": [

        ],
        "milestone": null,
        "comments": 1,
        "created_at": "2016-11-15T17:39:37Z",
        "updated_at": "2016-11-15T18:01:51Z",
        "closed_at": null,
        "pull_request": {
          "url": "https://api.github.com/repos/gopherjs/vecty/pulls/71",
          "html_url": "https://github.com/gopherjs/vecty/pull/71",
          "diff_url": "https://github.com/gopherjs/vecty/pull/71.diff",
          "patch_url": "https://github.com/gopherjs/vecty/pull/71.patch"
        },
        "body": "@slimsag "
      },
      "comment": {
        "url": "https://api.github.com/repos/gopherjs/vecty/issues/comments/260717425",
        "html_url": "https://github.com/gopherjs/vecty/pull/71#issuecomment-260717425",
        "issue_url": "https://api.github.com/repos/gopherjs/vecty/issues/71",
        "id": 260717425,
        "user": {
          "login": "shurcooL",
          "id": 1924134,
          "avatar_url": "https://avatars.githubusercontent.com/u/1924134?v=3",
          "gravatar_id": "",
          "url": "https://api.github.com/users/shurcooL",
          "html_url": "https://github.com/shurcooL",
          "followers_url": "https://api.github.com/users/shurcooL/followers",
          "following_url": "https://api.github.com/users/shurcooL/following{/other_user}",
          "gists_url": "https://api.github.com/users/shurcooL/gists{/gist_id}",
          "starred_url": "https://api.github.com/users/shurcooL/starred{/owner}{/repo}",
          "subscriptions_url": "https://api.github.com/users/shurcooL/subscriptions",
          "organizations_url": "https://api.github.com/users/shurcooL/orgs",
          "repos_url": "https://api.github.com/users/shurcooL/repos",
          "events_url": "https://api.github.com/users/shurcooL/events{/privacy}",
          "received_events_url": "https://api.github.com/users/shurcooL/received_events",
          "type": "User",
          "site_admin": false
        },
        "created_at": "2016-11-15T18:01:51Z",
        "updated_at": "2016-11-15T18:01:51Z",
        "body": "One style suggestion, but otherwise this is a nice and valid fix. LGTM."
      }
    },
    "public": true,
    "created_at": "2016-11-15T18:01:51Z",
    "org": {
      "id": 6654647,
      "login": "gopherjs",
      "gravatar_id": "",
      "url": "https://api.github.com/orgs/gopherjs",
      "avatar_url": "https://avatars.githubusercontent.com/u/6654647?"
    }
  },
  {
    "id": "4871401082",
    "type": "PullRequestReviewCommentEvent",
    "actor": {
      "id": 1924134,
      "login": "shurcooL",
      "display_login": "shurcooL",
      "gravatar_id": "",
      "url": "https://api.github.com/users/shurcooL",
      "avatar_url": "https://avatars.githubusercontent.com/u/1924134?"
    },
    "repo": {
      "id": 26097793,
      "name": "gopherjs/vecty",
      "url": "https://api.github.com/repos/gopherjs/vecty"
    },
    "payload": {
      "action": "created",
      "comment": {
        "url": "https://api.github.com/repos/gopherjs/vecty/pulls/comments/88074965",
        "pull_request_review_id": 8659906,
        "id": 88074965,
        "diff_hunk": "@@ -161,7 +161,8 @@ func (h *HTML) restoreHTML(prev *HTML) {\n \n // Restore implements the Restorer interface.\n func (h *HTML) Restore(old ComponentOrHTML) {\n-\tfor _, l := range h.eventListeners {\n+\tfor _, lrange := range h.eventListeners {\n+\t\tl := lrange",
        "path": "dom.go",
        "position": 6,
        "original_position": 6,
        "commit_id": "971e1dc8083d8aa7ba5c616869f9f53b791e84a7",
        "original_commit_id": "971e1dc8083d8aa7ba5c616869f9f53b791e84a7",
        "user": {
          "login": "shurcooL",
          "id": 1924134,
          "avatar_url": "https://avatars.githubusercontent.com/u/1924134?v=3",
          "gravatar_id": "",
          "url": "https://api.github.com/users/shurcooL",
          "html_url": "https://github.com/shurcooL",
          "followers_url": "https://api.github.com/users/shurcooL/followers",
          "following_url": "https://api.github.com/users/shurcooL/following{/other_user}",
          "gists_url": "https://api.github.com/users/shurcooL/gists{/gist_id}",
          "starred_url": "https://api.github.com/users/shurcooL/starred{/owner}{/repo}",
          "subscriptions_url": "https://api.github.com/users/shurcooL/subscriptions",
          "organizations_url": "https://api.github.com/users/shurcooL/orgs",
          "repos_url": "https://api.github.com/users/shurcooL/repos",
          "events_url": "https://api.github.com/users/shurcooL/events{/privacy}",
          "received_events_url": "https://api.github.com/users/shurcooL/received_events",
          "type": "User",
          "site_admin": false
        },
        "body": "Why not just ` + "`" + `l := l` + "`" + `? It's a well known pattern, and the idiomatic way of doing this.\r\n\r\n> It may seem odd to write\r\n>\r\n> ` + "```" + `Go\r\n> req := req\r\n> ` + "```" + `\r\n>\r\n> but it's legal and idiomatic in Go to do this. You get a fresh version of the variable with the same name, deliberately shadowing the loop variable locally but unique to each goroutine.\r\n\r\nSource: https://golang.org/doc/effective_go.html#channels\r\n\r\nAlso, ` + "`" + `lrange` + "`" + ` violates the [Mixed Caps](https://github.com/golang/go/wiki/CodeReviewComments#mixed-caps) naming convention, it would be ` + "`" + `lRange` + "`" + `.",
        "created_at": "2016-11-15T17:59:05Z",
        "updated_at": "2016-11-15T17:59:05Z",
        "html_url": "https://github.com/gopherjs/vecty/pull/71#discussion_r88074965",
        "pull_request_url": "https://api.github.com/repos/gopherjs/vecty/pulls/71",
        "_links": {
          "self": {
            "href": "https://api.github.com/repos/gopherjs/vecty/pulls/comments/88074965"
          },
          "html": {
            "href": "https://github.com/gopherjs/vecty/pull/71#discussion_r88074965"
          },
          "pull_request": {
            "href": "https://api.github.com/repos/gopherjs/vecty/pulls/71"
          }
        }
      },
      "pull_request": {
        "url": "https://api.github.com/repos/gopherjs/vecty/pulls/71",
        "id": 93813181,
        "html_url": "https://github.com/gopherjs/vecty/pull/71",
        "diff_url": "https://github.com/gopherjs/vecty/pull/71.diff",
        "patch_url": "https://github.com/gopherjs/vecty/pull/71.patch",
        "issue_url": "https://api.github.com/repos/gopherjs/vecty/issues/71",
        "number": 71,
        "state": "open",
        "locked": false,
        "title": "Prevent reuse of range variable in closure",
        "user": {
          "login": "davelondon",
          "id": 925351,
          "avatar_url": "https://avatars.githubusercontent.com/u/925351?v=3",
          "gravatar_id": "",
          "url": "https://api.github.com/users/davelondon",
          "html_url": "https://github.com/davelondon",
          "followers_url": "https://api.github.com/users/davelondon/followers",
          "following_url": "https://api.github.com/users/davelondon/following{/other_user}",
          "gists_url": "https://api.github.com/users/davelondon/gists{/gist_id}",
          "starred_url": "https://api.github.com/users/davelondon/starred{/owner}{/repo}",
          "subscriptions_url": "https://api.github.com/users/davelondon/subscriptions",
          "organizations_url": "https://api.github.com/users/davelondon/orgs",
          "repos_url": "https://api.github.com/users/davelondon/repos",
          "events_url": "https://api.github.com/users/davelondon/events{/privacy}",
          "received_events_url": "https://api.github.com/users/davelondon/received_events",
          "type": "User",
          "site_admin": false
        },
        "body": "@slimsag ",
        "created_at": "2016-11-15T17:39:37Z",
        "updated_at": "2016-11-15T17:59:05Z",
        "closed_at": null,
        "merged_at": null,
        "merge_commit_sha": "61bf6580eee651888c96305a5aeb247e5dc07621",
        "assignee": null,
        "assignees": [

        ],
        "milestone": null,
        "commits_url": "https://api.github.com/repos/gopherjs/vecty/pulls/71/commits",
        "review_comments_url": "https://api.github.com/repos/gopherjs/vecty/pulls/71/comments",
        "review_comment_url": "https://api.github.com/repos/gopherjs/vecty/pulls/comments{/number}",
        "comments_url": "https://api.github.com/repos/gopherjs/vecty/issues/71/comments",
        "statuses_url": "https://api.github.com/repos/gopherjs/vecty/statuses/971e1dc8083d8aa7ba5c616869f9f53b791e84a7",
        "head": {
          "label": "davelondon:patch-8",
          "ref": "patch-8",
          "sha": "971e1dc8083d8aa7ba5c616869f9f53b791e84a7",
          "user": {
            "login": "davelondon",
            "id": 925351,
            "avatar_url": "https://avatars.githubusercontent.com/u/925351?v=3",
            "gravatar_id": "",
            "url": "https://api.github.com/users/davelondon",
            "html_url": "https://github.com/davelondon",
            "followers_url": "https://api.github.com/users/davelondon/followers",
            "following_url": "https://api.github.com/users/davelondon/following{/other_user}",
            "gists_url": "https://api.github.com/users/davelondon/gists{/gist_id}",
            "starred_url": "https://api.github.com/users/davelondon/starred{/owner}{/repo}",
            "subscriptions_url": "https://api.github.com/users/davelondon/subscriptions",
            "organizations_url": "https://api.github.com/users/davelondon/orgs",
            "repos_url": "https://api.github.com/users/davelondon/repos",
            "events_url": "https://api.github.com/users/davelondon/events{/privacy}",
            "received_events_url": "https://api.github.com/users/davelondon/received_events",
            "type": "User",
            "site_admin": false
          },
          "repo": {
            "id": 54827624,
            "name": "vecty",
            "full_name": "davelondon/vecty",
            "owner": {
              "login": "davelondon",
              "id": 925351,
              "avatar_url": "https://avatars.githubusercontent.com/u/925351?v=3",
              "gravatar_id": "",
              "url": "https://api.github.com/users/davelondon",
              "html_url": "https://github.com/davelondon",
              "followers_url": "https://api.github.com/users/davelondon/followers",
              "following_url": "https://api.github.com/users/davelondon/following{/other_user}",
              "gists_url": "https://api.github.com/users/davelondon/gists{/gist_id}",
              "starred_url": "https://api.github.com/users/davelondon/starred{/owner}{/repo}",
              "subscriptions_url": "https://api.github.com/users/davelondon/subscriptions",
              "organizations_url": "https://api.github.com/users/davelondon/orgs",
              "repos_url": "https://api.github.com/users/davelondon/repos",
              "events_url": "https://api.github.com/users/davelondon/events{/privacy}",
              "received_events_url": "https://api.github.com/users/davelondon/received_events",
              "type": "User",
              "site_admin": false
            },
            "private": false,
            "html_url": "https://github.com/davelondon/vecty",
            "description": "Vecty: a frontend toolkit for Go",
            "fork": true,
            "url": "https://api.github.com/repos/davelondon/vecty",
            "forks_url": "https://api.github.com/repos/davelondon/vecty/forks",
            "keys_url": "https://api.github.com/repos/davelondon/vecty/keys{/key_id}",
            "collaborators_url": "https://api.github.com/repos/davelondon/vecty/collaborators{/collaborator}",
            "teams_url": "https://api.github.com/repos/davelondon/vecty/teams",
            "hooks_url": "https://api.github.com/repos/davelondon/vecty/hooks",
            "issue_events_url": "https://api.github.com/repos/davelondon/vecty/issues/events{/number}",
            "events_url": "https://api.github.com/repos/davelondon/vecty/events",
            "assignees_url": "https://api.github.com/repos/davelondon/vecty/assignees{/user}",
            "branches_url": "https://api.github.com/repos/davelondon/vecty/branches{/branch}",
            "tags_url": "https://api.github.com/repos/davelondon/vecty/tags",
            "blobs_url": "https://api.github.com/repos/davelondon/vecty/git/blobs{/sha}",
            "git_tags_url": "https://api.github.com/repos/davelondon/vecty/git/tags{/sha}",
            "git_refs_url": "https://api.github.com/repos/davelondon/vecty/git/refs{/sha}",
            "trees_url": "https://api.github.com/repos/davelondon/vecty/git/trees{/sha}",
            "statuses_url": "https://api.github.com/repos/davelondon/vecty/statuses/{sha}",
            "languages_url": "https://api.github.com/repos/davelondon/vecty/languages",
            "stargazers_url": "https://api.github.com/repos/davelondon/vecty/stargazers",
            "contributors_url": "https://api.github.com/repos/davelondon/vecty/contributors",
            "subscribers_url": "https://api.github.com/repos/davelondon/vecty/subscribers",
            "subscription_url": "https://api.github.com/repos/davelondon/vecty/subscription",
            "commits_url": "https://api.github.com/repos/davelondon/vecty/commits{/sha}",
            "git_commits_url": "https://api.github.com/repos/davelondon/vecty/git/commits{/sha}",
            "comments_url": "https://api.github.com/repos/davelondon/vecty/comments{/number}",
            "issue_comment_url": "https://api.github.com/repos/davelondon/vecty/issues/comments{/number}",
            "contents_url": "https://api.github.com/repos/davelondon/vecty/contents/{+path}",
            "compare_url": "https://api.github.com/repos/davelondon/vecty/compare/{base}...{head}",
            "merges_url": "https://api.github.com/repos/davelondon/vecty/merges",
            "archive_url": "https://api.github.com/repos/davelondon/vecty/{archive_format}{/ref}",
            "downloads_url": "https://api.github.com/repos/davelondon/vecty/downloads",
            "issues_url": "https://api.github.com/repos/davelondon/vecty/issues{/number}",
            "pulls_url": "https://api.github.com/repos/davelondon/vecty/pulls{/number}",
            "milestones_url": "https://api.github.com/repos/davelondon/vecty/milestones{/number}",
            "notifications_url": "https://api.github.com/repos/davelondon/vecty/notifications{?since,all,participating}",
            "labels_url": "https://api.github.com/repos/davelondon/vecty/labels{/name}",
            "releases_url": "https://api.github.com/repos/davelondon/vecty/releases{/id}",
            "deployments_url": "https://api.github.com/repos/davelondon/vecty/deployments",
            "created_at": "2016-03-27T12:34:11Z",
            "updated_at": "2016-03-27T12:34:12Z",
            "pushed_at": "2016-11-15T17:38:19Z",
            "git_url": "git://github.com/davelondon/vecty.git",
            "ssh_url": "git@github.com:davelondon/vecty.git",
            "clone_url": "https://github.com/davelondon/vecty.git",
            "svn_url": "https://github.com/davelondon/vecty",
            "homepage": "",
            "size": 478,
            "stargazers_count": 0,
            "watchers_count": 0,
            "language": "Go",
            "has_issues": false,
            "has_downloads": true,
            "has_wiki": true,
            "has_pages": false,
            "forks_count": 0,
            "mirror_url": null,
            "open_issues_count": 0,
            "forks": 0,
            "open_issues": 0,
            "watchers": 0,
            "default_branch": "master"
          }
        },
        "base": {
          "label": "gopherjs:master",
          "ref": "master",
          "sha": "99ceb90ed6f953903e800e6322bdf85a6dc7dcea",
          "user": {
            "login": "gopherjs",
            "id": 6654647,
            "avatar_url": "https://avatars.githubusercontent.com/u/6654647?v=3",
            "gravatar_id": "",
            "url": "https://api.github.com/users/gopherjs",
            "html_url": "https://github.com/gopherjs",
            "followers_url": "https://api.github.com/users/gopherjs/followers",
            "following_url": "https://api.github.com/users/gopherjs/following{/other_user}",
            "gists_url": "https://api.github.com/users/gopherjs/gists{/gist_id}",
            "starred_url": "https://api.github.com/users/gopherjs/starred{/owner}{/repo}",
            "subscriptions_url": "https://api.github.com/users/gopherjs/subscriptions",
            "organizations_url": "https://api.github.com/users/gopherjs/orgs",
            "repos_url": "https://api.github.com/users/gopherjs/repos",
            "events_url": "https://api.github.com/users/gopherjs/events{/privacy}",
            "received_events_url": "https://api.github.com/users/gopherjs/received_events",
            "type": "Organization",
            "site_admin": false
          },
          "repo": {
            "id": 26097793,
            "name": "vecty",
            "full_name": "gopherjs/vecty",
            "owner": {
              "login": "gopherjs",
              "id": 6654647,
              "avatar_url": "https://avatars.githubusercontent.com/u/6654647?v=3",
              "gravatar_id": "",
              "url": "https://api.github.com/users/gopherjs",
              "html_url": "https://github.com/gopherjs",
              "followers_url": "https://api.github.com/users/gopherjs/followers",
              "following_url": "https://api.github.com/users/gopherjs/following{/other_user}",
              "gists_url": "https://api.github.com/users/gopherjs/gists{/gist_id}",
              "starred_url": "https://api.github.com/users/gopherjs/starred{/owner}{/repo}",
              "subscriptions_url": "https://api.github.com/users/gopherjs/subscriptions",
              "organizations_url": "https://api.github.com/users/gopherjs/orgs",
              "repos_url": "https://api.github.com/users/gopherjs/repos",
              "events_url": "https://api.github.com/users/gopherjs/events{/privacy}",
              "received_events_url": "https://api.github.com/users/gopherjs/received_events",
              "type": "Organization",
              "site_admin": false
            },
            "private": false,
            "html_url": "https://github.com/gopherjs/vecty",
            "description": "Vecty: a frontend toolkit for GopherJS",
            "fork": false,
            "url": "https://api.github.com/repos/gopherjs/vecty",
            "forks_url": "https://api.github.com/repos/gopherjs/vecty/forks",
            "keys_url": "https://api.github.com/repos/gopherjs/vecty/keys{/key_id}",
            "collaborators_url": "https://api.github.com/repos/gopherjs/vecty/collaborators{/collaborator}",
            "teams_url": "https://api.github.com/repos/gopherjs/vecty/teams",
            "hooks_url": "https://api.github.com/repos/gopherjs/vecty/hooks",
            "issue_events_url": "https://api.github.com/repos/gopherjs/vecty/issues/events{/number}",
            "events_url": "https://api.github.com/repos/gopherjs/vecty/events",
            "assignees_url": "https://api.github.com/repos/gopherjs/vecty/assignees{/user}",
            "branches_url": "https://api.github.com/repos/gopherjs/vecty/branches{/branch}",
            "tags_url": "https://api.github.com/repos/gopherjs/vecty/tags",
            "blobs_url": "https://api.github.com/repos/gopherjs/vecty/git/blobs{/sha}",
            "git_tags_url": "https://api.github.com/repos/gopherjs/vecty/git/tags{/sha}",
            "git_refs_url": "https://api.github.com/repos/gopherjs/vecty/git/refs{/sha}",
            "trees_url": "https://api.github.com/repos/gopherjs/vecty/git/trees{/sha}",
            "statuses_url": "https://api.github.com/repos/gopherjs/vecty/statuses/{sha}",
            "languages_url": "https://api.github.com/repos/gopherjs/vecty/languages",
            "stargazers_url": "https://api.github.com/repos/gopherjs/vecty/stargazers",
            "contributors_url": "https://api.github.com/repos/gopherjs/vecty/contributors",
            "subscribers_url": "https://api.github.com/repos/gopherjs/vecty/subscribers",
            "subscription_url": "https://api.github.com/repos/gopherjs/vecty/subscription",
            "commits_url": "https://api.github.com/repos/gopherjs/vecty/commits{/sha}",
            "git_commits_url": "https://api.github.com/repos/gopherjs/vecty/git/commits{/sha}",
            "comments_url": "https://api.github.com/repos/gopherjs/vecty/comments{/number}",
            "issue_comment_url": "https://api.github.com/repos/gopherjs/vecty/issues/comments{/number}",
            "contents_url": "https://api.github.com/repos/gopherjs/vecty/contents/{+path}",
            "compare_url": "https://api.github.com/repos/gopherjs/vecty/compare/{base}...{head}",
            "merges_url": "https://api.github.com/repos/gopherjs/vecty/merges",
            "archive_url": "https://api.github.com/repos/gopherjs/vecty/{archive_format}{/ref}",
            "downloads_url": "https://api.github.com/repos/gopherjs/vecty/downloads",
            "issues_url": "https://api.github.com/repos/gopherjs/vecty/issues{/number}",
            "pulls_url": "https://api.github.com/repos/gopherjs/vecty/pulls{/number}",
            "milestones_url": "https://api.github.com/repos/gopherjs/vecty/milestones{/number}",
            "notifications_url": "https://api.github.com/repos/gopherjs/vecty/notifications{?since,all,participating}",
            "labels_url": "https://api.github.com/repos/gopherjs/vecty/labels{/name}",
            "releases_url": "https://api.github.com/repos/gopherjs/vecty/releases{/id}",
            "deployments_url": "https://api.github.com/repos/gopherjs/vecty/deployments",
            "created_at": "2014-11-03T00:32:48Z",
            "updated_at": "2016-11-14T20:44:31Z",
            "pushed_at": "2016-11-15T17:39:37Z",
            "git_url": "git://github.com/gopherjs/vecty.git",
            "ssh_url": "git@github.com:gopherjs/vecty.git",
            "clone_url": "https://github.com/gopherjs/vecty.git",
            "svn_url": "https://github.com/gopherjs/vecty",
            "homepage": "",
            "size": 468,
            "stargazers_count": 96,
            "watchers_count": 96,
            "language": "Go",
            "has_issues": true,
            "has_downloads": true,
            "has_wiki": true,
            "has_pages": true,
            "forks_count": 9,
            "mirror_url": null,
            "open_issues_count": 19,
            "forks": 9,
            "open_issues": 19,
            "watchers": 96,
            "default_branch": "master"
          }
        },
        "_links": {
          "self": {
            "href": "https://api.github.com/repos/gopherjs/vecty/pulls/71"
          },
          "html": {
            "href": "https://github.com/gopherjs/vecty/pull/71"
          },
          "issue": {
            "href": "https://api.github.com/repos/gopherjs/vecty/issues/71"
          },
          "comments": {
            "href": "https://api.github.com/repos/gopherjs/vecty/issues/71/comments"
          },
          "review_comments": {
            "href": "https://api.github.com/repos/gopherjs/vecty/pulls/71/comments"
          },
          "review_comment": {
            "href": "https://api.github.com/repos/gopherjs/vecty/pulls/comments{/number}"
          },
          "commits": {
            "href": "https://api.github.com/repos/gopherjs/vecty/pulls/71/commits"
          },
          "statuses": {
            "href": "https://api.github.com/repos/gopherjs/vecty/statuses/971e1dc8083d8aa7ba5c616869f9f53b791e84a7"
          }
        }
      }
    },
    "public": true,
    "created_at": "2016-11-15T17:59:05Z",
    "org": {
      "id": 6654647,
      "login": "gopherjs",
      "gravatar_id": "",
      "url": "https://api.github.com/orgs/gopherjs",
      "avatar_url": "https://avatars.githubusercontent.com/u/6654647?"
    }
  },
  {
    "id": "4868242658",
    "type": "IssueCommentEvent",
    "actor": {
      "id": 1924134,
      "login": "shurcooL",
      "display_login": "shurcooL",
      "gravatar_id": "",
      "url": "https://api.github.com/users/shurcooL",
      "avatar_url": "https://avatars.githubusercontent.com/u/1924134?"
    },
    "repo": {
      "id": 73430589,
      "name": "google/gops",
      "url": "https://api.github.com/repos/google/gops"
    },
    "payload": {
      "action": "created",
      "issue": {
        "url": "https://api.github.com/repos/google/gops/issues/10",
        "repository_url": "https://api.github.com/repos/google/gops",
        "labels_url": "https://api.github.com/repos/google/gops/issues/10/labels{/name}",
        "comments_url": "https://api.github.com/repos/google/gops/issues/10/comments",
        "events_url": "https://api.github.com/repos/google/gops/issues/10/events",
        "html_url": "https://github.com/google/gops/issues/10",
        "id": 189211421,
        "number": 10,
        "title": "Consider some way to configure unix socket path",
        "user": {
          "login": "wolfeidau",
          "id": 50636,
          "avatar_url": "https://avatars.githubusercontent.com/u/50636?v=3",
          "gravatar_id": "",
          "url": "https://api.github.com/users/wolfeidau",
          "html_url": "https://github.com/wolfeidau",
          "followers_url": "https://api.github.com/users/wolfeidau/followers",
          "following_url": "https://api.github.com/users/wolfeidau/following{/other_user}",
          "gists_url": "https://api.github.com/users/wolfeidau/gists{/gist_id}",
          "starred_url": "https://api.github.com/users/wolfeidau/starred{/owner}{/repo}",
          "subscriptions_url": "https://api.github.com/users/wolfeidau/subscriptions",
          "organizations_url": "https://api.github.com/users/wolfeidau/orgs",
          "repos_url": "https://api.github.com/users/wolfeidau/repos",
          "events_url": "https://api.github.com/users/wolfeidau/events{/privacy}",
          "received_events_url": "https://api.github.com/users/wolfeidau/received_events",
          "type": "User",
          "site_admin": false
        },
        "labels": [

        ],
        "state": "closed",
        "locked": false,
        "assignee": null,
        "assignees": [

        ],
        "milestone": null,
        "comments": 5,
        "created_at": "2016-11-14T20:04:06Z",
        "updated_at": "2016-11-15T08:44:18Z",
        "closed_at": "2016-11-15T06:59:10Z",
        "body": "Just reviewing the code and would love a way to move that socket out of ` + "`" + `/tmp` + "`" + ` to say ` + "`" + `/var/run` + "`" + ` for services. Reason I mention it is the ` + "`" + `PrivateTmp` + "`" + ` feature provided by systemd.\r\n\r\nMaybe an environment variable would be the simplest way to override?\r\n\r\nCheers"
      },
      "comment": {
        "url": "https://api.github.com/repos/google/gops/issues/comments/260581250",
        "html_url": "https://github.com/google/gops/issues/10#issuecomment-260581250",
        "issue_url": "https://api.github.com/repos/google/gops/issues/10",
        "id": 260581250,
        "user": {
          "login": "shurcooL",
          "id": 1924134,
          "avatar_url": "https://avatars.githubusercontent.com/u/1924134?v=3",
          "gravatar_id": "",
          "url": "https://api.github.com/users/shurcooL",
          "html_url": "https://github.com/shurcooL",
          "followers_url": "https://api.github.com/users/shurcooL/followers",
          "following_url": "https://api.github.com/users/shurcooL/following{/other_user}",
          "gists_url": "https://api.github.com/users/shurcooL/gists{/gist_id}",
          "starred_url": "https://api.github.com/users/shurcooL/starred{/owner}{/repo}",
          "subscriptions_url": "https://api.github.com/users/shurcooL/subscriptions",
          "organizations_url": "https://api.github.com/users/shurcooL/orgs",
          "repos_url": "https://api.github.com/users/shurcooL/repos",
          "events_url": "https://api.github.com/users/shurcooL/events{/privacy}",
          "received_events_url": "https://api.github.com/users/shurcooL/received_events",
          "type": "User",
          "site_admin": false
        },
        "created_at": "2016-11-15T08:44:18Z",
        "updated_at": "2016-11-15T08:44:18Z",
        "body": "> I want to move off of the unix socket and want to listen on a TCP socket instead.\r\n\r\nI'm asking because I don't know and would like to learn, what's the reason for you wanting to do that?\r\n\r\nYou mentioned this would \"also\" enable windows support, so it sounds like that's not the primary reason. Are there other reasons?"
      }
    },
    "public": true,
    "created_at": "2016-11-15T08:44:18Z",
    "org": {
      "id": 1342004,
      "login": "google",
      "gravatar_id": "",
      "url": "https://api.github.com/orgs/google",
      "avatar_url": "https://avatars.githubusercontent.com/u/1342004?"
    }
  },
  {
    "id": "4860320046",
    "type": "PushEvent",
    "actor": {
      "id": 1924134,
      "login": "shurcooL",
      "display_login": "shurcooL",
      "gravatar_id": "",
      "url": "https://api.github.com/users/shurcooL",
      "avatar_url": "https://avatars.githubusercontent.com/u/1924134?"
    },
    "repo": {
      "id": 18283571,
      "name": "shurcooL/markdownfmt",
      "url": "https://api.github.com/repos/shurcooL/markdownfmt"
    },
    "payload": {
      "push_id": 1400576707,
      "size": 1,
      "distinct_size": 1,
      "ref": "refs/heads/master",
      "head": "ba769786099144f93b987583ecc09a22a8e04040",
      "before": "e4c5ae7b0021d524158ada2365bb9b270e010635",
      "commits": [
        {
          "sha": "ba769786099144f93b987583ecc09a22a8e04040",
          "author": {
            "email": "shurcooL@gmail.com",
            "name": "Dmitri Shuralyov"
          },
          "message": "Don't use terminal styling when listing or diffing.\n\nIt has adverse effects on the diff, and is useless when listing files\nonly.",
          "distinct": true,
          "url": "https://api.github.com/repos/shurcooL/markdownfmt/commits/ba769786099144f93b987583ecc09a22a8e04040"
        }
      ]
    },
    "public": true,
    "created_at": "2016-11-14T01:35:55Z"
  },
  {
    "id": "4860272206",
    "type": "PushEvent",
    "actor": {
      "id": 1924134,
      "login": "shurcooL",
      "display_login": "shurcooL",
      "gravatar_id": "",
      "url": "https://api.github.com/users/shurcooL",
      "avatar_url": "https://avatars.githubusercontent.com/u/1924134?"
    },
    "repo": {
      "id": 55930476,
      "name": "shurcooL/home",
      "url": "https://api.github.com/repos/shurcooL/home"
    },
    "payload": {
      "push_id": 1400558876,
      "size": 1,
      "distinct_size": 1,
      "ref": "refs/heads/master",
      "head": "c570ef7b7aa7dec1cf3941b005d47b943b73b3d8",
      "before": "69fd2187c02d5f1b2200f42243ed31a346345689",
      "commits": [
        {
          "sha": "c570ef7b7aa7dec1cf3941b005d47b943b73b3d8",
          "author": {
            "email": "shurcooL@gmail.com",
            "name": "Dmitri Shuralyov"
          },
          "message": "assets: Regenerate.\n\nIncludes update in shurcooL/resume@d563d8ff299b1a46d6cb1a00de3fdb3f1e76b7b3.",
          "distinct": true,
          "url": "https://api.github.com/repos/shurcooL/home/commits/c570ef7b7aa7dec1cf3941b005d47b943b73b3d8"
        }
      ]
    },
    "public": true,
    "created_at": "2016-11-14T01:11:33Z"
  },
  {
    "id": "4860242004",
    "type": "PushEvent",
    "actor": {
      "id": 1924134,
      "login": "shurcooL",
      "display_login": "shurcooL",
      "gravatar_id": "",
      "url": "https://api.github.com/users/shurcooL",
      "avatar_url": "https://avatars.githubusercontent.com/u/1924134?"
    },
    "repo": {
      "id": 55930469,
      "name": "shurcooL/resume",
      "url": "https://api.github.com/repos/shurcooL/resume"
    },
    "payload": {
      "push_id": 1400547631,
      "size": 1,
      "distinct_size": 1,
      "ref": "refs/heads/master",
      "head": "d563d8ff299b1a46d6cb1a00de3fdb3f1e76b7b3",
      "before": "906c671edb9030f7780faea686f0127273c5e15a",
      "commits": [
        {
          "sha": "d563d8ff299b1a46d6cb1a00de3fdb3f1e76b7b3",
          "author": {
            "email": "shurcooL@gmail.com",
            "name": "Dmitri Shuralyov"
          },
          "message": "Update Sourcegraph dates.\n\nIt has been an incredible journey!",
          "distinct": true,
          "url": "https://api.github.com/repos/shurcooL/resume/commits/d563d8ff299b1a46d6cb1a00de3fdb3f1e76b7b3"
        }
      ]
    },
    "public": true,
    "created_at": "2016-11-14T00:55:41Z"
  },
  {
    "id": "4860202414",
    "type": "IssueCommentEvent",
    "actor": {
      "id": 1924134,
      "login": "shurcooL",
      "display_login": "shurcooL",
      "gravatar_id": "",
      "url": "https://api.github.com/users/shurcooL",
      "avatar_url": "https://avatars.githubusercontent.com/u/1924134?"
    },
    "repo": {
      "id": 12418999,
      "name": "gopherjs/gopherjs",
      "url": "https://api.github.com/repos/gopherjs/gopherjs"
    },
    "payload": {
      "action": "created",
      "issue": {
        "url": "https://api.github.com/repos/gopherjs/gopherjs/issues/546",
        "repository_url": "https://api.github.com/repos/gopherjs/gopherjs",
        "labels_url": "https://api.github.com/repos/gopherjs/gopherjs/issues/546/labels{/name}",
        "comments_url": "https://api.github.com/repos/gopherjs/gopherjs/issues/546/comments",
        "events_url": "https://api.github.com/repos/gopherjs/gopherjs/issues/546/events",
        "html_url": "https://github.com/gopherjs/gopherjs/issues/546",
        "id": 187514325,
        "number": 546,
        "title": "Recursion Error in Iceweasel 49, trying to run the gopherjs/jquery example in README.md",
        "user": {
          "login": "ZenifiedFromI2P",
          "id": 22223903,
          "avatar_url": "https://avatars.githubusercontent.com/u/22223903?v=3",
          "gravatar_id": "",
          "url": "https://api.github.com/users/ZenifiedFromI2P",
          "html_url": "https://github.com/ZenifiedFromI2P",
          "followers_url": "https://api.github.com/users/ZenifiedFromI2P/followers",
          "following_url": "https://api.github.com/users/ZenifiedFromI2P/following{/other_user}",
          "gists_url": "https://api.github.com/users/ZenifiedFromI2P/gists{/gist_id}",
          "starred_url": "https://api.github.com/users/ZenifiedFromI2P/starred{/owner}{/repo}",
          "subscriptions_url": "https://api.github.com/users/ZenifiedFromI2P/subscriptions",
          "organizations_url": "https://api.github.com/users/ZenifiedFromI2P/orgs",
          "repos_url": "https://api.github.com/users/ZenifiedFromI2P/repos",
          "events_url": "https://api.github.com/users/ZenifiedFromI2P/events{/privacy}",
          "received_events_url": "https://api.github.com/users/ZenifiedFromI2P/received_events",
          "type": "User",
          "site_admin": false
        },
        "labels": [

        ],
        "state": "open",
        "locked": false,
        "assignee": null,
        "assignees": [

        ],
        "milestone": null,
        "comments": 13,
        "created_at": "2016-11-05T18:28:40Z",
        "updated_at": "2016-11-14T00:32:47Z",
        "closed_at": null,
        "body": "I tried to run the jquery example in gopherjs/jquery, in my Iceweasel 49 (Firefox), note I have NoScript and few privacy addons enabled (uBlock Origin, HTTPS Everywhere, etc.)..\r\n\r\n` + "```" + `\r\nYour current jQuery version is: 2.1.0\r\ntoo much recursion <Learn More> main.js:int:int\r\n` + "```" + `\r\ngo env (It may be useless):\r\n` + "```" + `\r\nGOARCH=\"amd64\"\r\nGOBIN=\"\"\r\nGOEXE=\"\"\r\nGOHOSTARCH=\"amd64\"\r\nGOHOSTOS=\"linux\"\r\nGOOS=\"linux\"\r\nGOPATH=\"/home/user/gopath\"\r\nGORACE=\"\"\r\nGOROOT=\"/usr/lib/go\"\r\nGOTOOLDIR=\"/usr/lib/go/pkg/tool/linux_amd64\"\r\nCC=\"gcc\"\r\nGOGCCFLAGS=\"-fPIC -m64 -pthread -fmessage-length=0 -fdebug-prefix-map=/tmp/go-build875613271=/tmp/go-build -gno-record-gcc-switches\"\r\nCXX=\"g++\"\r\nCGO_ENABLED=\"1\"\r\n` + "```" + `\r\nEdit: Found out more after debugging:\r\n\r\nIt issues at main.js:2059:1, main.js is here: https://bpaste.net/raw/9652fed5cdcc\r\n\r\n\r\n"
      },
      "comment": {
        "url": "https://api.github.com/repos/gopherjs/gopherjs/issues/comments/260225099",
        "html_url": "https://github.com/gopherjs/gopherjs/issues/546#issuecomment-260225099",
        "issue_url": "https://api.github.com/repos/gopherjs/gopherjs/issues/546",
        "id": 260225099,
        "user": {
          "login": "shurcooL",
          "id": 1924134,
          "avatar_url": "https://avatars.githubusercontent.com/u/1924134?v=3",
          "gravatar_id": "",
          "url": "https://api.github.com/users/shurcooL",
          "html_url": "https://github.com/shurcooL",
          "followers_url": "https://api.github.com/users/shurcooL/followers",
          "following_url": "https://api.github.com/users/shurcooL/following{/other_user}",
          "gists_url": "https://api.github.com/users/shurcooL/gists{/gist_id}",
          "starred_url": "https://api.github.com/users/shurcooL/starred{/owner}{/repo}",
          "subscriptions_url": "https://api.github.com/users/shurcooL/subscriptions",
          "organizations_url": "https://api.github.com/users/shurcooL/orgs",
          "repos_url": "https://api.github.com/users/shurcooL/repos",
          "events_url": "https://api.github.com/users/shurcooL/events{/privacy}",
          "received_events_url": "https://api.github.com/users/shurcooL/received_events",
          "type": "User",
          "site_admin": false
        },
        "created_at": "2016-11-14T00:32:47Z",
        "updated_at": "2016-11-14T00:32:47Z",
        "body": "I tried the ` + "`" + `ThisTestFails` + "`" + ` snippet you posted, and not seeing an issue there.\r\n\r\n![image](https://cloud.githubusercontent.com/assets/1924134/20250130/b460064e-a9be-11e6-9bcf-5782f282ab89.png)\r\n\r\nCan you verify you have the latest version of GopherJS? Try updating it with ` + "`" + `go get -u -v github.com/gopherjs/gopherjs` + "`" + `. If you have latest version, that command should not print anything."
      }
    },
    "public": true,
    "created_at": "2016-11-14T00:32:47Z",
    "org": {
      "id": 6654647,
      "login": "gopherjs",
      "gravatar_id": "",
      "url": "https://api.github.com/orgs/gopherjs",
      "avatar_url": "https://avatars.githubusercontent.com/u/6654647?"
    }
  },
  {
    "id": "4860195622",
    "type": "IssueCommentEvent",
    "actor": {
      "id": 1924134,
      "login": "shurcooL",
      "display_login": "shurcooL",
      "gravatar_id": "",
      "url": "https://api.github.com/users/shurcooL",
      "avatar_url": "https://avatars.githubusercontent.com/u/1924134?"
    },
    "repo": {
      "id": 12418999,
      "name": "gopherjs/gopherjs",
      "url": "https://api.github.com/repos/gopherjs/gopherjs"
    },
    "payload": {
      "action": "created",
      "issue": {
        "url": "https://api.github.com/repos/gopherjs/gopherjs/issues/546",
        "repository_url": "https://api.github.com/repos/gopherjs/gopherjs",
        "labels_url": "https://api.github.com/repos/gopherjs/gopherjs/issues/546/labels{/name}",
        "comments_url": "https://api.github.com/repos/gopherjs/gopherjs/issues/546/comments",
        "events_url": "https://api.github.com/repos/gopherjs/gopherjs/issues/546/events",
        "html_url": "https://github.com/gopherjs/gopherjs/issues/546",
        "id": 187514325,
        "number": 546,
        "title": "Recursion Error in Iceweasel 49, trying to run the gopherjs/jquery example in README.md",
        "user": {
          "login": "ZenifiedFromI2P",
          "id": 22223903,
          "avatar_url": "https://avatars.githubusercontent.com/u/22223903?v=3",
          "gravatar_id": "",
          "url": "https://api.github.com/users/ZenifiedFromI2P",
          "html_url": "https://github.com/ZenifiedFromI2P",
          "followers_url": "https://api.github.com/users/ZenifiedFromI2P/followers",
          "following_url": "https://api.github.com/users/ZenifiedFromI2P/following{/other_user}",
          "gists_url": "https://api.github.com/users/ZenifiedFromI2P/gists{/gist_id}",
          "starred_url": "https://api.github.com/users/ZenifiedFromI2P/starred{/owner}{/repo}",
          "subscriptions_url": "https://api.github.com/users/ZenifiedFromI2P/subscriptions",
          "organizations_url": "https://api.github.com/users/ZenifiedFromI2P/orgs",
          "repos_url": "https://api.github.com/users/ZenifiedFromI2P/repos",
          "events_url": "https://api.github.com/users/ZenifiedFromI2P/events{/privacy}",
          "received_events_url": "https://api.github.com/users/ZenifiedFromI2P/received_events",
          "type": "User",
          "site_admin": false
        },
        "labels": [

        ],
        "state": "open",
        "locked": false,
        "assignee": null,
        "assignees": [

        ],
        "milestone": null,
        "comments": 11,
        "created_at": "2016-11-05T18:28:40Z",
        "updated_at": "2016-11-14T00:29:31Z",
        "closed_at": null,
        "body": "I tried to run the jquery example in gopherjs/jquery, in my Iceweasel 49 (Firefox), note I have NoScript and few privacy addons enabled (uBlock Origin, HTTPS Everywhere, etc.)..\r\n\r\n` + "```" + `\r\nYour current jQuery version is: 2.1.0\r\ntoo much recursion <Learn More> main.js:int:int\r\n` + "```" + `\r\ngo env (It may be useless):\r\n` + "```" + `\r\nGOARCH=\"amd64\"\r\nGOBIN=\"\"\r\nGOEXE=\"\"\r\nGOHOSTARCH=\"amd64\"\r\nGOHOSTOS=\"linux\"\r\nGOOS=\"linux\"\r\nGOPATH=\"/home/user/gopath\"\r\nGORACE=\"\"\r\nGOROOT=\"/usr/lib/go\"\r\nGOTOOLDIR=\"/usr/lib/go/pkg/tool/linux_amd64\"\r\nCC=\"gcc\"\r\nGOGCCFLAGS=\"-fPIC -m64 -pthread -fmessage-length=0 -fdebug-prefix-map=/tmp/go-build875613271=/tmp/go-build -gno-record-gcc-switches\"\r\nCXX=\"g++\"\r\nCGO_ENABLED=\"1\"\r\n` + "```" + `\r\nEdit: Found out more after debugging:\r\n\r\nIt issues at main.js:2059:1, main.js is here: https://bpaste.net/raw/9652fed5cdcc\r\n\r\n\r\n"
      },
      "comment": {
        "url": "https://api.github.com/repos/gopherjs/gopherjs/issues/comments/260224893",
        "html_url": "https://github.com/gopherjs/gopherjs/issues/546#issuecomment-260224893",
        "issue_url": "https://api.github.com/repos/gopherjs/gopherjs/issues/546",
        "id": 260224893,
        "user": {
          "login": "shurcooL",
          "id": 1924134,
          "avatar_url": "https://avatars.githubusercontent.com/u/1924134?v=3",
          "gravatar_id": "",
          "url": "https://api.github.com/users/shurcooL",
          "html_url": "https://github.com/shurcooL",
          "followers_url": "https://api.github.com/users/shurcooL/followers",
          "following_url": "https://api.github.com/users/shurcooL/following{/other_user}",
          "gists_url": "https://api.github.com/users/shurcooL/gists{/gist_id}",
          "starred_url": "https://api.github.com/users/shurcooL/starred{/owner}{/repo}",
          "subscriptions_url": "https://api.github.com/users/shurcooL/subscriptions",
          "organizations_url": "https://api.github.com/users/shurcooL/orgs",
          "repos_url": "https://api.github.com/users/shurcooL/repos",
          "events_url": "https://api.github.com/users/shurcooL/events{/privacy}",
          "received_events_url": "https://api.github.com/users/shurcooL/received_events",
          "type": "User",
          "site_admin": false
        },
        "created_at": "2016-11-14T00:29:31Z",
        "updated_at": "2016-11-14T00:29:31Z",
        "body": "Thanks for posting the HTML, I am able to reproduce with that. When I type some text in the input box, I get \"Uncaught RangeError: Maximum call stack size exceeded\" errors:\r\n\r\n![image](https://cloud.githubusercontent.com/assets/1924134/20250118/2f868f10-a9be-11e6-8a25-12046a842ad8.png)\r\n\r\nIt looks like a programming error to me. There is infinite recursion:\r\n\r\n![image](https://cloud.githubusercontent.com/assets/1924134/20250121/4a1cca06-a9be-11e6-96c0-7cdad80fae92.png)"
      }
    },
    "public": true,
    "created_at": "2016-11-14T00:29:32Z",
    "org": {
      "id": 6654647,
      "login": "gopherjs",
      "gravatar_id": "",
      "url": "https://api.github.com/orgs/gopherjs",
      "avatar_url": "https://avatars.githubusercontent.com/u/6654647?"
    }
  },
  {
    "id": "4860178537",
    "type": "IssuesEvent",
    "actor": {
      "id": 1924134,
      "login": "shurcooL",
      "display_login": "shurcooL",
      "gravatar_id": "",
      "url": "https://api.github.com/users/shurcooL",
      "avatar_url": "https://avatars.githubusercontent.com/u/1924134?"
    },
    "repo": {
      "id": 25803522,
      "name": "rakyll/gom",
      "url": "https://api.github.com/repos/rakyll/gom"
    },
    "payload": {
      "action": "opened",
      "issue": {
        "url": "https://api.github.com/repos/rakyll/gom/issues/22",
        "repository_url": "https://api.github.com/repos/rakyll/gom",
        "labels_url": "https://api.github.com/repos/rakyll/gom/issues/22/labels{/name}",
        "comments_url": "https://api.github.com/repos/rakyll/gom/issues/22/comments",
        "events_url": "https://api.github.com/repos/rakyll/gom/issues/22/events",
        "html_url": "https://github.com/rakyll/gom/issues/22",
        "id": 189006348,
        "number": 22,
        "title": "README: Broken screenshot.",
        "user": {
          "login": "shurcooL",
          "id": 1924134,
          "avatar_url": "https://avatars.githubusercontent.com/u/1924134?v=3",
          "gravatar_id": "",
          "url": "https://api.github.com/users/shurcooL",
          "html_url": "https://github.com/shurcooL",
          "followers_url": "https://api.github.com/users/shurcooL/followers",
          "following_url": "https://api.github.com/users/shurcooL/following{/other_user}",
          "gists_url": "https://api.github.com/users/shurcooL/gists{/gist_id}",
          "starred_url": "https://api.github.com/users/shurcooL/starred{/owner}{/repo}",
          "subscriptions_url": "https://api.github.com/users/shurcooL/subscriptions",
          "organizations_url": "https://api.github.com/users/shurcooL/orgs",
          "repos_url": "https://api.github.com/users/shurcooL/repos",
          "events_url": "https://api.github.com/users/shurcooL/events{/privacy}",
          "received_events_url": "https://api.github.com/users/shurcooL/received_events",
          "type": "User",
          "site_admin": false
        },
        "labels": [

        ],
        "state": "open",
        "locked": false,
        "assignee": null,
        "assignees": [

        ],
        "milestone": null,
        "comments": 0,
        "created_at": "2016-11-14T00:19:38Z",
        "updated_at": "2016-11-14T00:19:38Z",
        "closed_at": null,
        "body": "https://googledrive.com/host/0ByfSjdPVs9MZbkhjeUhMYzRTeEE/gom-screenshot.png is 404.\r\n\r\n![image](https://cloud.githubusercontent.com/assets/1924134/20250061/e7b8c604-a9bc-11e6-8cfe-93da47c955f6.png)"
      }
    },
    "public": true,
    "created_at": "2016-11-14T00:19:38Z"
  },
  {
    "id": "4860175191",
    "type": "IssueCommentEvent",
    "actor": {
      "id": 1924134,
      "login": "shurcooL",
      "display_login": "shurcooL",
      "gravatar_id": "",
      "url": "https://api.github.com/users/shurcooL",
      "avatar_url": "https://avatars.githubusercontent.com/u/1924134?"
    },
    "repo": {
      "id": 32201174,
      "name": "abyssoft/teleport",
      "url": "https://api.github.com/repos/abyssoft/teleport"
    },
    "payload": {
      "action": "created",
      "issue": {
        "url": "https://api.github.com/repos/abyssoft/teleport/issues/34",
        "repository_url": "https://api.github.com/repos/abyssoft/teleport",
        "labels_url": "https://api.github.com/repos/abyssoft/teleport/issues/34/labels{/name}",
        "comments_url": "https://api.github.com/repos/abyssoft/teleport/issues/34/comments",
        "events_url": "https://api.github.com/repos/abyssoft/teleport/issues/34/events",
        "html_url": "https://github.com/abyssoft/teleport/issues/34",
        "id": 188895041,
        "number": 34,
        "title": "What needs to happen to see better support for Teleport?",
        "user": {
          "login": "ylluminate",
          "id": 248440,
          "avatar_url": "https://avatars.githubusercontent.com/u/248440?v=3",
          "gravatar_id": "",
          "url": "https://api.github.com/users/ylluminate",
          "html_url": "https://github.com/ylluminate",
          "followers_url": "https://api.github.com/users/ylluminate/followers",
          "following_url": "https://api.github.com/users/ylluminate/following{/other_user}",
          "gists_url": "https://api.github.com/users/ylluminate/gists{/gist_id}",
          "starred_url": "https://api.github.com/users/ylluminate/starred{/owner}{/repo}",
          "subscriptions_url": "https://api.github.com/users/ylluminate/subscriptions",
          "organizations_url": "https://api.github.com/users/ylluminate/orgs",
          "repos_url": "https://api.github.com/users/ylluminate/repos",
          "events_url": "https://api.github.com/users/ylluminate/events{/privacy}",
          "received_events_url": "https://api.github.com/users/ylluminate/received_events",
          "type": "User",
          "site_admin": false
        },
        "labels": [

        ],
        "state": "open",
        "locked": false,
        "assignee": null,
        "assignees": [

        ],
        "milestone": null,
        "comments": 3,
        "created_at": "2016-11-12T06:08:47Z",
        "updated_at": "2016-11-14T00:17:33Z",
        "closed_at": null,
        "body": "Teleport is a fantastic little tool that beats the pants off of Synergy.  Do we need to establish a shareware / donation / commercial model for Teleport in order to see it start getting better support and more development?\r\n\r\n"
      },
      "comment": {
        "url": "https://api.github.com/repos/abyssoft/teleport/issues/comments/260224168",
        "html_url": "https://github.com/abyssoft/teleport/issues/34#issuecomment-260224168",
        "issue_url": "https://api.github.com/repos/abyssoft/teleport/issues/34",
        "id": 260224168,
        "user": {
          "login": "shurcooL",
          "id": 1924134,
          "avatar_url": "https://avatars.githubusercontent.com/u/1924134?v=3",
          "gravatar_id": "",
          "url": "https://api.github.com/users/shurcooL",
          "html_url": "https://github.com/shurcooL",
          "followers_url": "https://api.github.com/users/shurcooL/followers",
          "following_url": "https://api.github.com/users/shurcooL/following{/other_user}",
          "gists_url": "https://api.github.com/users/shurcooL/gists{/gist_id}",
          "starred_url": "https://api.github.com/users/shurcooL/starred{/owner}{/repo}",
          "subscriptions_url": "https://api.github.com/users/shurcooL/subscriptions",
          "organizations_url": "https://api.github.com/users/shurcooL/orgs",
          "repos_url": "https://api.github.com/users/shurcooL/repos",
          "events_url": "https://api.github.com/users/shurcooL/events{/privacy}",
          "received_events_url": "https://api.github.com/users/shurcooL/received_events",
          "type": "User",
          "site_admin": false
        },
        "created_at": "2016-11-14T00:17:33Z",
        "updated_at": "2016-11-14T00:17:33Z",
        "body": "A key thing to keep in mind is that Teleport supports macOS only. So it's only applicable for multiple Macs. Synergy and similar work (but nowhere near as well) under macOS, Linux, Windows."
      }
    },
    "public": true,
    "created_at": "2016-11-14T00:17:33Z"
  },
  {
    "id": "4860162151",
    "type": "IssueCommentEvent",
    "actor": {
      "id": 1924134,
      "login": "shurcooL",
      "display_login": "shurcooL",
      "gravatar_id": "",
      "url": "https://api.github.com/users/shurcooL",
      "avatar_url": "https://avatars.githubusercontent.com/u/1924134?"
    },
    "repo": {
      "id": 12418999,
      "name": "gopherjs/gopherjs",
      "url": "https://api.github.com/repos/gopherjs/gopherjs"
    },
    "payload": {
      "action": "created",
      "issue": {
        "url": "https://api.github.com/repos/gopherjs/gopherjs/issues/542",
        "repository_url": "https://api.github.com/repos/gopherjs/gopherjs",
        "labels_url": "https://api.github.com/repos/gopherjs/gopherjs/issues/542/labels{/name}",
        "comments_url": "https://api.github.com/repos/gopherjs/gopherjs/issues/542/comments",
        "events_url": "https://api.github.com/repos/gopherjs/gopherjs/issues/542/events",
        "html_url": "https://github.com/gopherjs/gopherjs/issues/542",
        "id": 184678638,
        "number": 542,
        "title": "Proposal: Implement ` + "`" + `js` + "`" + ` package with friendlier panics on unsupported architectures.",
        "user": {
          "login": "shurcooL",
          "id": 1924134,
          "avatar_url": "https://avatars.githubusercontent.com/u/1924134?v=3",
          "gravatar_id": "",
          "url": "https://api.github.com/users/shurcooL",
          "html_url": "https://github.com/shurcooL",
          "followers_url": "https://api.github.com/users/shurcooL/followers",
          "following_url": "https://api.github.com/users/shurcooL/following{/other_user}",
          "gists_url": "https://api.github.com/users/shurcooL/gists{/gist_id}",
          "starred_url": "https://api.github.com/users/shurcooL/starred{/owner}{/repo}",
          "subscriptions_url": "https://api.github.com/users/shurcooL/subscriptions",
          "organizations_url": "https://api.github.com/users/shurcooL/orgs",
          "repos_url": "https://api.github.com/users/shurcooL/repos",
          "events_url": "https://api.github.com/users/shurcooL/events{/privacy}",
          "received_events_url": "https://api.github.com/users/shurcooL/received_events",
          "type": "User",
          "site_admin": false
        },
        "labels": [

        ],
        "state": "open",
        "locked": false,
        "assignee": null,
        "assignees": [

        ],
        "milestone": null,
        "comments": 4,
        "created_at": "2016-10-23T08:49:35Z",
        "updated_at": "2016-11-14T00:10:07Z",
        "closed_at": null,
        "body": "Right now, the [` + "`" + `github.com/gopherjs/gopherjs/js` + "`" + ` package](https://godoc.org/github.com/gopherjs/gopherjs/js) is officially only implemented for [` + "`" + `js` + "`" + ` architecture](https://github.com/gopherjs/gopherjs#architecture). That's where using it is meaningful, as:\n\n> Package js provides functions for interacting with native JavaScript APIs. Calls to these functions are treated specially by GopherJS and translated directly to their corresponding JavaScript syntax.\n\nHowever, in reality, it is also currently implemented for all other architectures. The implementation panics for most things with unfriendly messages, such as \"invalid memory address or nil pointer dereference\".\n\nThis means someone who tries to run Go code that (directly or indirectly) uses ` + "`" + `js` + "`" + ` package under an unsupported architecture will likely see an unfriendly panic, such as:\n\n` + "```" + `\ntodomvc $ GOARCH=amd64 go run -compiler=gc example.go \npanic: runtime error: invalid memory address or nil pointer dereference\n[signal SIGSEGV: segmentation violation code=0x1 addr=0x0 pc=0x744f2]\n\ngoroutine 1 [running]:\npanic(0xcaa20, 0xc42000a090)\n    /usr/local/go/src/runtime/panic.go:500 +0x1a1\ngithub.com/gopherjs/gopherjs/js.(*Object).Get(0x0, 0xe66bf, 0xc, 0xf3788)\n    /Users/Dmitri/Dropbox/Work/2013/GoLand/src/github.com/gopherjs/gopherjs/js/js.go:32 +0x22\nmain.attachLocalStorage()\n    /Users/Dmitri/Dropbox/Work/2013/GoLand/src/github.com/gopherjs/vecty/examples/todomvc/example.go:39 +0x7c\nmain.main()\n    /Users/Dmitri/Dropbox/Work/2013/GoLand/src/github.com/gopherjs/vecty/examples/todomvc/example.go:17 +0x26\nexit status 2\n` + "```" + `\n\n**I propose we change implementation of ` + "`" + `github.com/gopherjs/gopherjs/js` + "`" + ` for unsupported architectures (everything other than ` + "`" + `js` + "`" + `) such that it panics with friendly error messages.** For example:\n\n` + "```" + `\ntodomvc $ GOARCH=amd64 go run -compiler=gc example.go \npanic: js.Object.Get is only implemented on js architecture, it's not implemented on amd64 architecture\n\ngoroutine 1 [running]:\npanic(0xc9340, 0xc42000a350)\n    /usr/local/go/src/runtime/panic.go:500 +0x1a1\ngithub.com/gopherjs/gopherjs/js.(*Object).Get(0x0, 0xe6764, 0xc, 0xf38a8)\n    /Users/Dmitri/Dropbox/Work/2013/GoLand/src/github.com/gopherjs/gopherjs/js/unimplemented.go:13 +0xe4\nmain.attachLocalStorage()\n    /Users/Dmitri/Dropbox/Work/2013/GoLand/src/github.com/gopherjs/vecty/examples/todomvc/example.go:39 +0x7c\nmain.main()\n    /Users/Dmitri/Dropbox/Work/2013/GoLand/src/github.com/gopherjs/vecty/examples/todomvc/example.go:17 +0x26\nexit status 2\n` + "```" + `\n\nI think that's a good thing to do, and it should help people less familiar with GopherJS understand what's going on.\n\nThis idea is inspired by previous experiences and, most recently, by https://github.com/gopherjs/vecty/issues/61. /cc @campoy\n### Implementation Discussion\n\nIn my prototype, I simply added ` + "`" + `// +build js` + "`" + ` constraint to [` + "`" + `js.go` + "`" + ` file](https://github.com/gopherjs/gopherjs/blob/master/js/js.go), and created a second file ` + "`" + `unimplemented.go` + "`" + ` that looks like this:\n\n` + "```" + ` Go\n// +build !js\n\npackage js\n\nimport (\n    \"fmt\"\n    \"runtime\"\n)\n\ntype Object struct{ object *Object }\n\nfunc (o *Object) Get(key string) *Object {\n    panic(fmt.Errorf(\"js.Object.Get is only implemented on js architecture, it's not implemented on %s architecture\", runtime.GOARCH))\n}\n\n// ...\n` + "```" + `\n\nHowever, the challenging part of the implementation is dealing with ` + "`" + `godoc` + "`" + `. When doing the above, it causes godoc of ` + "`" + `github.com/gopherjs/gopherjs/js` + "`" + ` package to show up without any documentation, because the ` + "`" + `amd64` + "`" + ` architecture has higher priority over ` + "`" + `js` + "`" + `.\n\nI can think of a couple alternative solutions, but so far I'm not sure what's the best one.\n\nBefore I spend more time thinking about this, I wanted to make the proposal to see what the GopherJS team thinks. /cc @neelance\n"
      },
      "comment": {
        "url": "https://api.github.com/repos/gopherjs/gopherjs/issues/comments/260223785",
        "html_url": "https://github.com/gopherjs/gopherjs/issues/542#issuecomment-260223785",
        "issue_url": "https://api.github.com/repos/gopherjs/gopherjs/issues/542",
        "id": 260223785,
        "user": {
          "login": "shurcooL",
          "id": 1924134,
          "avatar_url": "https://avatars.githubusercontent.com/u/1924134?v=3",
          "gravatar_id": "",
          "url": "https://api.github.com/users/shurcooL",
          "html_url": "https://github.com/shurcooL",
          "followers_url": "https://api.github.com/users/shurcooL/followers",
          "following_url": "https://api.github.com/users/shurcooL/following{/other_user}",
          "gists_url": "https://api.github.com/users/shurcooL/gists{/gist_id}",
          "starred_url": "https://api.github.com/users/shurcooL/starred{/owner}{/repo}",
          "subscriptions_url": "https://api.github.com/users/shurcooL/subscriptions",
          "organizations_url": "https://api.github.com/users/shurcooL/orgs",
          "repos_url": "https://api.github.com/users/shurcooL/repos",
          "events_url": "https://api.github.com/users/shurcooL/events{/privacy}",
          "received_events_url": "https://api.github.com/users/shurcooL/received_events",
          "type": "User",
          "site_admin": false
        },
        "created_at": "2016-11-14T00:10:07Z",
        "updated_at": "2016-11-14T00:10:07Z",
        "body": "@neelance, how do you feel about this?\r\n\r\n@alkchr, GopherJS started out with ` + "`" + `Object` + "`" + ` being an ` + "`" + `interface{}` + "`" + ` originally. However, it made the change from that to pointer to ` + "`" + `struct{}` + "`" + `, because it was a better representation. See https://github.com/gopherjs/gopherjs/commit/0853187ab4154e73cf6f7bd43625ab83e762a000#diff-37afcbb71b3321eb845150243b012cec."
      }
    },
    "public": true,
    "created_at": "2016-11-14T00:10:07Z",
    "org": {
      "id": 6654647,
      "login": "gopherjs",
      "gravatar_id": "",
      "url": "https://api.github.com/orgs/gopherjs",
      "avatar_url": "https://avatars.githubusercontent.com/u/6654647?"
    }
  },
  {
    "id": "4860146328",
    "type": "IssueCommentEvent",
    "actor": {
      "id": 1924134,
      "login": "shurcooL",
      "display_login": "shurcooL",
      "gravatar_id": "",
      "url": "https://api.github.com/users/shurcooL",
      "avatar_url": "https://avatars.githubusercontent.com/u/1924134?"
    },
    "repo": {
      "id": 10270722,
      "name": "google/go-github",
      "url": "https://api.github.com/repos/google/go-github"
    },
    "payload": {
      "action": "created",
      "issue": {
        "url": "https://api.github.com/repos/google/go-github/issues/472",
        "repository_url": "https://api.github.com/repos/google/go-github",
        "labels_url": "https://api.github.com/repos/google/go-github/issues/472/labels{/name}",
        "comments_url": "https://api.github.com/repos/google/go-github/issues/472/comments",
        "events_url": "https://api.github.com/repos/google/go-github/issues/472/events",
        "html_url": "https://github.com/google/go-github/pull/472",
        "id": 188935876,
        "number": 472,
        "title": "Update License struct and add new RepositoryLicense struct",
        "user": {
          "login": "nmiyake",
          "id": 4267425,
          "avatar_url": "https://avatars.githubusercontent.com/u/4267425?v=3",
          "gravatar_id": "",
          "url": "https://api.github.com/users/nmiyake",
          "html_url": "https://github.com/nmiyake",
          "followers_url": "https://api.github.com/users/nmiyake/followers",
          "following_url": "https://api.github.com/users/nmiyake/following{/other_user}",
          "gists_url": "https://api.github.com/users/nmiyake/gists{/gist_id}",
          "starred_url": "https://api.github.com/users/nmiyake/starred{/owner}{/repo}",
          "subscriptions_url": "https://api.github.com/users/nmiyake/subscriptions",
          "organizations_url": "https://api.github.com/users/nmiyake/orgs",
          "repos_url": "https://api.github.com/users/nmiyake/repos",
          "events_url": "https://api.github.com/users/nmiyake/events{/privacy}",
          "received_events_url": "https://api.github.com/users/nmiyake/received_events",
          "type": "User",
          "site_admin": false
        },
        "labels": [

        ],
        "state": "open",
        "locked": false,
        "assignee": null,
        "assignees": [

        ],
        "milestone": null,
        "comments": 3,
        "created_at": "2016-11-12T21:30:43Z",
        "updated_at": "2016-11-14T00:01:12Z",
        "closed_at": null,
        "pull_request": {
          "url": "https://api.github.com/repos/google/go-github/pulls/472",
          "html_url": "https://github.com/google/go-github/pull/472",
          "diff_url": "https://github.com/google/go-github/pull/472.diff",
          "patch_url": "https://github.com/google/go-github/pull/472.patch"
        },
        "body": "Update structs to reflect the new reponses returned from GitHub.\r\n\r\nFixes #471"
      },
      "comment": {
        "url": "https://api.github.com/repos/google/go-github/issues/comments/260223306",
        "html_url": "https://github.com/google/go-github/pull/472#issuecomment-260223306",
        "issue_url": "https://api.github.com/repos/google/go-github/issues/472",
        "id": 260223306,
        "user": {
          "login": "shurcooL",
          "id": 1924134,
          "avatar_url": "https://avatars.githubusercontent.com/u/1924134?v=3",
          "gravatar_id": "",
          "url": "https://api.github.com/users/shurcooL",
          "html_url": "https://github.com/shurcooL",
          "followers_url": "https://api.github.com/users/shurcooL/followers",
          "following_url": "https://api.github.com/users/shurcooL/following{/other_user}",
          "gists_url": "https://api.github.com/users/shurcooL/gists{/gist_id}",
          "starred_url": "https://api.github.com/users/shurcooL/starred{/owner}{/repo}",
          "subscriptions_url": "https://api.github.com/users/shurcooL/subscriptions",
          "organizations_url": "https://api.github.com/users/shurcooL/orgs",
          "repos_url": "https://api.github.com/users/shurcooL/repos",
          "events_url": "https://api.github.com/users/shurcooL/events{/privacy}",
          "received_events_url": "https://api.github.com/users/shurcooL/received_events",
          "type": "User",
          "site_admin": false
        },
        "created_at": "2016-11-14T00:01:12Z",
        "updated_at": "2016-11-14T00:01:12Z",
        "body": "As soon as @gmlewis or @willnorris or someone else reviews it and gives it a second LGTM."
      }
    },
    "public": true,
    "created_at": "2016-11-14T00:01:12Z",
    "org": {
      "id": 1342004,
      "login": "google",
      "gravatar_id": "",
      "url": "https://api.github.com/orgs/google",
      "avatar_url": "https://avatars.githubusercontent.com/u/1342004?"
    }
  },
  {
    "id": "4860128460",
    "type": "IssuesEvent",
    "actor": {
      "id": 1924134,
      "login": "shurcooL",
      "display_login": "shurcooL",
      "gravatar_id": "",
      "url": "https://api.github.com/users/shurcooL",
      "avatar_url": "https://avatars.githubusercontent.com/u/1924134?"
    },
    "repo": {
      "id": 9524997,
      "name": "glfw/glfw",
      "url": "https://api.github.com/repos/glfw/glfw"
    },
    "payload": {
      "action": "closed",
      "issue": {
        "url": "https://api.github.com/repos/glfw/glfw/issues/896",
        "repository_url": "https://api.github.com/repos/glfw/glfw",
        "labels_url": "https://api.github.com/repos/glfw/glfw/issues/896/labels{/name}",
        "comments_url": "https://api.github.com/repos/glfw/glfw/issues/896/comments",
        "events_url": "https://api.github.com/repos/glfw/glfw/issues/896/events",
        "html_url": "https://github.com/glfw/glfw/issues/896",
        "id": 188957358,
        "number": 896,
        "title": "VS CODE + GO + GLFW + DELVE CANT COMPILE",
        "user": {
          "login": "MrLiet",
          "id": 23429917,
          "avatar_url": "https://avatars.githubusercontent.com/u/23429917?v=3",
          "gravatar_id": "",
          "url": "https://api.github.com/users/MrLiet",
          "html_url": "https://github.com/MrLiet",
          "followers_url": "https://api.github.com/users/MrLiet/followers",
          "following_url": "https://api.github.com/users/MrLiet/following{/other_user}",
          "gists_url": "https://api.github.com/users/MrLiet/gists{/gist_id}",
          "starred_url": "https://api.github.com/users/MrLiet/starred{/owner}{/repo}",
          "subscriptions_url": "https://api.github.com/users/MrLiet/subscriptions",
          "organizations_url": "https://api.github.com/users/MrLiet/orgs",
          "repos_url": "https://api.github.com/users/MrLiet/repos",
          "events_url": "https://api.github.com/users/MrLiet/events{/privacy}",
          "received_events_url": "https://api.github.com/users/MrLiet/received_events",
          "type": "User",
          "site_admin": false
        },
        "labels": [

        ],
        "state": "closed",
        "locked": false,
        "assignee": null,
        "assignees": [

        ],
        "milestone": null,
        "comments": 3,
        "created_at": "2016-11-13T07:42:08Z",
        "updated_at": "2016-11-13T23:51:06Z",
        "closed_at": "2016-11-13T23:51:06Z",
        "body": "Hello.\r\n\r\nI have a problem:\r\n\r\n![image](https://cloud.githubusercontent.com/assets/23429917/20244303/9d7a465c-a9ae-11e6-8e94-2a7773b65bd2.png)\r\n\r\nSource code:\r\n\r\n` + "```" + `\r\npackage main\r\n\r\nimport \"github.com/go-gl/glfw/v3.2/glfw\"\r\n\r\nfunc main() {\r\n\r\n\tglfw.Init()\r\n\r\n\tw, _ := glfw.CreateWindow(800, 600, \"hello\", nil, nil)\r\n\tvar running bool = true\r\n\tfor running {\r\n\t\tglfw.PollEvents()\r\n\t\tif w.ShouldClose() {\r\n\t\t\trunning = false\r\n\t\t}\r\n\t}\r\n\r\n}\r\n\r\n` + "```" + `\r\n\r\nConsole output:\r\n\r\n> # me/proj1\r\n> github.com/go-gl/glfw/v3.2/glfw(.text): strdup: not defined\r\n> github.com/go-gl/glfw/v3.2/glfw(.text): strdup: not defined\r\n> github.com/go-gl/glfw/v3.2/glfw(.text): strdup: not defined\r\n> github.com/go-gl/glfw/v3.2/glfw(.text): strdup: not defined\r\n> github.com/go-gl/glfw/v3.2/glfw(.text): undefined: strdup\r\n> github.com/go-gl/glfw/v3.2/glfw(.text): undefined: strdup\r\n> github.com/go-gl/glfw/v3.2/glfw(.text): undefined: strdup\r\n> github.com/go-gl/glfw/v3.2/glfw(.text): undefined: strdup\r\n> exit status 2\r\n> \r\n\r\nIm use:\r\nWindows 64 bit\r\nGo 1.7.3\r\nMingw 64\r\nVS Code\r\n\r\nVS Code settings:\r\n{\r\n    \"go.buildOnSave\": true,\r\n    \"go.lintOnSave\": true,\r\n    \"go.vetOnSave\": true,\r\n    \"go.buildTags\": \"\",\r\n    \"go.buildFlags\": [],\r\n    \"go.lintTool\": \"golint\",\r\n    \"go.lintFlags\": [],\r\n    \"go.vetFlags\": [],\r\n    \"go.coverOnSave\":false,\r\n    \"go.useCodeSnippetsOnFunctionSuggest\": true,\r\n    \"go.formatOnSave\": true, \r\n    \"go.formatTool\": \"goreturns\",\r\n    \"go.formatFlags\": [],\r\n    \"go.gocodeAutoBuild\": false,\r\n    \"go.autocompleteUnimportedPackages\": true\r\n    \r\n    \r\n}\r\n\r\nand Launch settings:\r\n{\r\n    \"version\": \"0.2.0\",\r\n    \"configurations\": [\r\n        {\r\n            \"name\": \"Launch\",\r\n            \"type\": \"go\",\r\n            \"request\": \"launch\",\r\n            \"mode\": \"debug\",\r\n            \"program\": \"${workspaceRoot}\",\r\n            \"env\": {},\r\n            \"args\": []\r\n        }\r\n    ]\r\n}\r\n\r\nHelp!"
      }
    },
    "public": true,
    "created_at": "2016-11-13T23:51:06Z",
    "org": {
      "id": 3905364,
      "login": "glfw",
      "gravatar_id": "",
      "url": "https://api.github.com/orgs/glfw",
      "avatar_url": "https://avatars.githubusercontent.com/u/3905364?"
    }
  },
  {
    "id": "4860124075",
    "type": "IssueCommentEvent",
    "actor": {
      "id": 1924134,
      "login": "shurcooL",
      "display_login": "shurcooL",
      "gravatar_id": "",
      "url": "https://api.github.com/users/shurcooL",
      "avatar_url": "https://avatars.githubusercontent.com/u/1924134?"
    },
    "repo": {
      "id": 9524997,
      "name": "glfw/glfw",
      "url": "https://api.github.com/repos/glfw/glfw"
    },
    "payload": {
      "action": "created",
      "issue": {
        "url": "https://api.github.com/repos/glfw/glfw/issues/896",
        "repository_url": "https://api.github.com/repos/glfw/glfw",
        "labels_url": "https://api.github.com/repos/glfw/glfw/issues/896/labels{/name}",
        "comments_url": "https://api.github.com/repos/glfw/glfw/issues/896/comments",
        "events_url": "https://api.github.com/repos/glfw/glfw/issues/896/events",
        "html_url": "https://github.com/glfw/glfw/issues/896",
        "id": 188957358,
        "number": 896,
        "title": "VS CODE + GO + GLFW + DELVE CANT COMPILE",
        "user": {
          "login": "MrLiet",
          "id": 23429917,
          "avatar_url": "https://avatars.githubusercontent.com/u/23429917?v=3",
          "gravatar_id": "",
          "url": "https://api.github.com/users/MrLiet",
          "html_url": "https://github.com/MrLiet",
          "followers_url": "https://api.github.com/users/MrLiet/followers",
          "following_url": "https://api.github.com/users/MrLiet/following{/other_user}",
          "gists_url": "https://api.github.com/users/MrLiet/gists{/gist_id}",
          "starred_url": "https://api.github.com/users/MrLiet/starred{/owner}{/repo}",
          "subscriptions_url": "https://api.github.com/users/MrLiet/subscriptions",
          "organizations_url": "https://api.github.com/users/MrLiet/orgs",
          "repos_url": "https://api.github.com/users/MrLiet/repos",
          "events_url": "https://api.github.com/users/MrLiet/events{/privacy}",
          "received_events_url": "https://api.github.com/users/MrLiet/received_events",
          "type": "User",
          "site_admin": false
        },
        "labels": [

        ],
        "state": "open",
        "locked": false,
        "assignee": null,
        "assignees": [

        ],
        "milestone": null,
        "comments": 3,
        "created_at": "2016-11-13T07:42:08Z",
        "updated_at": "2016-11-13T23:48:14Z",
        "closed_at": null,
        "body": "Hello.\r\n\r\nI have a problem:\r\n\r\n![image](https://cloud.githubusercontent.com/assets/23429917/20244303/9d7a465c-a9ae-11e6-8e94-2a7773b65bd2.png)\r\n\r\nSource code:\r\n\r\n` + "```" + `\r\npackage main\r\n\r\nimport \"github.com/go-gl/glfw/v3.2/glfw\"\r\n\r\nfunc main() {\r\n\r\n\tglfw.Init()\r\n\r\n\tw, _ := glfw.CreateWindow(800, 600, \"hello\", nil, nil)\r\n\tvar running bool = true\r\n\tfor running {\r\n\t\tglfw.PollEvents()\r\n\t\tif w.ShouldClose() {\r\n\t\t\trunning = false\r\n\t\t}\r\n\t}\r\n\r\n}\r\n\r\n` + "```" + `\r\n\r\nConsole output:\r\n\r\n> # me/proj1\r\n> github.com/go-gl/glfw/v3.2/glfw(.text): strdup: not defined\r\n> github.com/go-gl/glfw/v3.2/glfw(.text): strdup: not defined\r\n> github.com/go-gl/glfw/v3.2/glfw(.text): strdup: not defined\r\n> github.com/go-gl/glfw/v3.2/glfw(.text): strdup: not defined\r\n> github.com/go-gl/glfw/v3.2/glfw(.text): undefined: strdup\r\n> github.com/go-gl/glfw/v3.2/glfw(.text): undefined: strdup\r\n> github.com/go-gl/glfw/v3.2/glfw(.text): undefined: strdup\r\n> github.com/go-gl/glfw/v3.2/glfw(.text): undefined: strdup\r\n> exit status 2\r\n> \r\n\r\nIm use:\r\nWindows 64 bit\r\nGo 1.7.3\r\nMingw 64\r\nVS Code\r\n\r\nVS Code settings:\r\n{\r\n    \"go.buildOnSave\": true,\r\n    \"go.lintOnSave\": true,\r\n    \"go.vetOnSave\": true,\r\n    \"go.buildTags\": \"\",\r\n    \"go.buildFlags\": [],\r\n    \"go.lintTool\": \"golint\",\r\n    \"go.lintFlags\": [],\r\n    \"go.vetFlags\": [],\r\n    \"go.coverOnSave\":false,\r\n    \"go.useCodeSnippetsOnFunctionSuggest\": true,\r\n    \"go.formatOnSave\": true, \r\n    \"go.formatTool\": \"goreturns\",\r\n    \"go.formatFlags\": [],\r\n    \"go.gocodeAutoBuild\": false,\r\n    \"go.autocompleteUnimportedPackages\": true\r\n    \r\n    \r\n}\r\n\r\nand Launch settings:\r\n{\r\n    \"version\": \"0.2.0\",\r\n    \"configurations\": [\r\n        {\r\n            \"name\": \"Launch\",\r\n            \"type\": \"go\",\r\n            \"request\": \"launch\",\r\n            \"mode\": \"debug\",\r\n            \"program\": \"${workspaceRoot}\",\r\n            \"env\": {},\r\n            \"args\": []\r\n        }\r\n    ]\r\n}\r\n\r\nHelp!"
      },
      "comment": {
        "url": "https://api.github.com/repos/glfw/glfw/issues/comments/260222633",
        "html_url": "https://github.com/glfw/glfw/issues/896#issuecomment-260222633",
        "issue_url": "https://api.github.com/repos/glfw/glfw/issues/896",
        "id": 260222633,
        "user": {
          "login": "shurcooL",
          "id": 1924134,
          "avatar_url": "https://avatars.githubusercontent.com/u/1924134?v=3",
          "gravatar_id": "",
          "url": "https://api.github.com/users/shurcooL",
          "html_url": "https://github.com/shurcooL",
          "followers_url": "https://api.github.com/users/shurcooL/followers",
          "following_url": "https://api.github.com/users/shurcooL/following{/other_user}",
          "gists_url": "https://api.github.com/users/shurcooL/gists{/gist_id}",
          "starred_url": "https://api.github.com/users/shurcooL/starred{/owner}{/repo}",
          "subscriptions_url": "https://api.github.com/users/shurcooL/subscriptions",
          "organizations_url": "https://api.github.com/users/shurcooL/orgs",
          "repos_url": "https://api.github.com/users/shurcooL/repos",
          "events_url": "https://api.github.com/users/shurcooL/events{/privacy}",
          "received_events_url": "https://api.github.com/users/shurcooL/received_events",
          "type": "User",
          "site_admin": false
        },
        "created_at": "2016-11-13T23:48:14Z",
        "updated_at": "2016-11-13T23:48:14Z",
        "body": "@MrLiet, this is the wrong issue tracker to report this issue. It seems to be completely related to the Go bindings for GLFW, not GLFW the C library itself. You should report it at:\r\n\r\nhttps://github.com/go-gl/glfw/issues"
      }
    },
    "public": true,
    "created_at": "2016-11-13T23:48:15Z",
    "org": {
      "id": 3905364,
      "login": "glfw",
      "gravatar_id": "",
      "url": "https://api.github.com/orgs/glfw",
      "avatar_url": "https://avatars.githubusercontent.com/u/3905364?"
    }
  },
  {
    "id": "4860115712",
    "type": "IssueCommentEvent",
    "actor": {
      "id": 1924134,
      "login": "shurcooL",
      "display_login": "shurcooL",
      "gravatar_id": "",
      "url": "https://api.github.com/users/shurcooL",
      "avatar_url": "https://avatars.githubusercontent.com/u/1924134?"
    },
    "repo": {
      "id": 23096959,
      "name": "golang/go",
      "url": "https://api.github.com/repos/golang/go"
    },
    "payload": {
      "action": "created",
      "issue": {
        "url": "https://api.github.com/repos/golang/go/issues/17780",
        "repository_url": "https://api.github.com/repos/golang/go",
        "labels_url": "https://api.github.com/repos/golang/go/issues/17780/labels{/name}",
        "comments_url": "https://api.github.com/repos/golang/go/issues/17780/comments",
        "events_url": "https://api.github.com/repos/golang/go/issues/17780/events",
        "html_url": "https://github.com/golang/go/issues/17780",
        "id": 187171559,
        "number": 17780,
        "title": "cmd/vet: detect \"defer resp.Body.Close()\"s that will panic",
        "user": {
          "login": "ahmetalpbalkan",
          "id": 159209,
          "avatar_url": "https://avatars.githubusercontent.com/u/159209?v=3",
          "gravatar_id": "",
          "url": "https://api.github.com/users/ahmetalpbalkan",
          "html_url": "https://github.com/ahmetalpbalkan",
          "followers_url": "https://api.github.com/users/ahmetalpbalkan/followers",
          "following_url": "https://api.github.com/users/ahmetalpbalkan/following{/other_user}",
          "gists_url": "https://api.github.com/users/ahmetalpbalkan/gists{/gist_id}",
          "starred_url": "https://api.github.com/users/ahmetalpbalkan/starred{/owner}{/repo}",
          "subscriptions_url": "https://api.github.com/users/ahmetalpbalkan/subscriptions",
          "organizations_url": "https://api.github.com/users/ahmetalpbalkan/orgs",
          "repos_url": "https://api.github.com/users/ahmetalpbalkan/repos",
          "events_url": "https://api.github.com/users/ahmetalpbalkan/events{/privacy}",
          "received_events_url": "https://api.github.com/users/ahmetalpbalkan/received_events",
          "type": "User",
          "site_admin": false
        },
        "labels": [
          {
            "id": 236419512,
            "url": "https://api.github.com/repos/golang/go/labels/Proposal",
            "name": "Proposal",
            "color": "ededed",
            "default": false
          },
          {
            "id": 246350233,
            "url": "https://api.github.com/repos/golang/go/labels/Proposal-Accepted",
            "name": "Proposal-Accepted",
            "color": "009800",
            "default": false
          }
        ],
        "state": "closed",
        "locked": false,
        "assignee": null,
        "assignees": [

        ],
        "milestone": null,
        "comments": 16,
        "created_at": "2016-11-03T19:47:24Z",
        "updated_at": "2016-11-13T23:43:18Z",
        "closed_at": "2016-11-10T20:38:26Z",
        "body": "This is a simple newbie mistake I saw in the Go code I wrote many years ago:\r\n\r\n` + "```" + `go\r\nresp, err := http.Get(...)\r\ndefer resp.Body.Close()\r\nif err != nil {\r\n    return err\r\n}\r\n// read resp.Body or something\r\n` + "```" + `\r\n\r\nThis will panic when ` + "`" + `resp == nil` + "`" + ` which is _iff_ ` + "`" + `err != nil` + "`" + `. Can ` + "`" + `go vet` + "`" + ` have a heuristic for when ` + "`" + `resp.Body.Close()` + "`" + ` is invoked in a path where neither of the following are checked:\r\n\r\n- ` + "`" + `resp != nil` + "`" + `\r\n- ` + "`" + `err != nil` + "`" + `\r\n\r\nI don't know the specifics of how cmd/vet works but I was a little surprised when no static analysis tools caught this one so far (obviously points to lack of testing on my end as well, however I feel like static analysis could've found it earlier)."
      },
      "comment": {
        "url": "https://api.github.com/repos/golang/go/issues/comments/260222350",
        "html_url": "https://github.com/golang/go/issues/17780#issuecomment-260222350",
        "issue_url": "https://api.github.com/repos/golang/go/issues/17780",
        "id": 260222350,
        "user": {
          "login": "shurcooL",
          "id": 1924134,
          "avatar_url": "https://avatars.githubusercontent.com/u/1924134?v=3",
          "gravatar_id": "",
          "url": "https://api.github.com/users/shurcooL",
          "html_url": "https://github.com/shurcooL",
          "followers_url": "https://api.github.com/users/shurcooL/followers",
          "following_url": "https://api.github.com/users/shurcooL/following{/other_user}",
          "gists_url": "https://api.github.com/users/shurcooL/gists{/gist_id}",
          "starred_url": "https://api.github.com/users/shurcooL/starred{/owner}{/repo}",
          "subscriptions_url": "https://api.github.com/users/shurcooL/subscriptions",
          "organizations_url": "https://api.github.com/users/shurcooL/orgs",
          "repos_url": "https://api.github.com/users/shurcooL/repos",
          "events_url": "https://api.github.com/users/shurcooL/events{/privacy}",
          "received_events_url": "https://api.github.com/users/shurcooL/received_events",
          "type": "User",
          "site_admin": false
        },
        "created_at": "2016-11-13T23:43:18Z",
        "updated_at": "2016-11-13T23:43:18Z",
        "body": "> but I was a little surprised when no static analysis tools caught this one so far\r\n\r\n@ahmetalpbalkan, FYI, I know that [` + "`" + `staticcheck` + "`" + `](https://github.com/dominikh/go-staticcheck#checks) does catch this issue (and similar ones, like with ` + "`" + `os.Open` + "`" + `, etc.).\r\n\r\n> - Don't ` + "`" + `defer rc.Close()` + "`" + ` before having checked the error returned by ` + "`" + `Open` + "`" + ` or similar.\r\n\r\nFor example, if I run it on the following program:\r\n\r\n` + "```" + `Go\r\npackage main\r\n\r\nimport (\r\n\t\"fmt\"\r\n\t\"io\"\r\n\t\"io/ioutil\"\r\n\t\"net/http\"\r\n)\r\n\r\nfunc main() {\r\n\tresp, err := http.Get(\"https://www.example.com/\")\r\n\tdefer resp.Body.Close()\r\n\tif err != nil {\r\n\t\tfmt.Println(err)\r\n\t\treturn\r\n\t}\r\n\t// read resp.Body or something\r\n\tio.Copy(ioutil.Discard, resp.Body)\r\n}\r\n` + "```" + `\r\n\r\nThe output is:\r\n\r\n` + "```" + `\r\n$ staticcheck \r\nmain.go:12:2: should check returned error before deferring resp.Body.Close()\r\n` + "```" + `"
      }
    },
    "public": true,
    "created_at": "2016-11-13T23:43:20Z",
    "org": {
      "id": 4314092,
      "login": "golang",
      "gravatar_id": "",
      "url": "https://api.github.com/orgs/golang",
      "avatar_url": "https://avatars.githubusercontent.com/u/4314092?"
    }
  },
  {
    "id": "4860085539",
    "type": "PullRequestReviewCommentEvent",
    "actor": {
      "id": 1924134,
      "login": "shurcooL",
      "display_login": "shurcooL",
      "gravatar_id": "",
      "url": "https://api.github.com/users/shurcooL",
      "avatar_url": "https://avatars.githubusercontent.com/u/1924134?"
    },
    "repo": {
      "id": 62099451,
      "name": "campoy/embedmd",
      "url": "https://api.github.com/repos/campoy/embedmd"
    },
    "payload": {
      "action": "created",
      "comment": {
        "url": "https://api.github.com/repos/campoy/embedmd/pulls/comments/87725064",
        "pull_request_review_id": 8325288,
        "id": 87725064,
        "diff_hunk": "@@ -0,0 +1,48 @@\n+// Copyright 2016 Google Inc. All rights reserved.\n+// Licensed under the Apache License, Version 2.0 (the \"License\");\n+// you may not use this file except in compliance with the License.\n+// You may obtain a copy of the License at\n+// http://www.apache.org/licenses/LICENSE-2.0\n+//\n+// Unless required by applicable law or agreed to writing, software distributed\n+// under the License is distributed on a \"AS IS\" BASIS, WITHOUT WARRANTIES OR\n+// CONDITIONS OF ANY KIND, either express or implied.\n+//\n+// See the License for the specific language governing permissions and\n+// limitations under the License.\n+\n+package embedmd\n+\n+import (\n+\t\"fmt\"\n+\t\"io/ioutil\"\n+\t\"net/http\"\n+\t\"path/filepath\"\n+\t\"strings\"\n+)\n+\n+// Fetcher provides an abstraction on a file system.\n+// The Fetch function is called anytime some content needs to be fetched.\n+// For now this includes files and URLs.\n+type Fetcher interface {\n+\tFetch(dir, path string) ([]byte, error)",
        "path": "embedmd/content.go",
        "position": 28,
        "original_position": 28,
        "commit_id": "7e3080602f24bb83fcd0ce194538f1e1cedb7cfc",
        "original_commit_id": "7e3080602f24bb83fcd0ce194538f1e1cedb7cfc",
        "user": {
          "login": "shurcooL",
          "id": 1924134,
          "avatar_url": "https://avatars.githubusercontent.com/u/1924134?v=3",
          "gravatar_id": "",
          "url": "https://api.github.com/users/shurcooL",
          "html_url": "https://github.com/shurcooL",
          "followers_url": "https://api.github.com/users/shurcooL/followers",
          "following_url": "https://api.github.com/users/shurcooL/following{/other_user}",
          "gists_url": "https://api.github.com/users/shurcooL/gists{/gist_id}",
          "starred_url": "https://api.github.com/users/shurcooL/starred{/owner}{/repo}",
          "subscriptions_url": "https://api.github.com/users/shurcooL/subscriptions",
          "organizations_url": "https://api.github.com/users/shurcooL/orgs",
          "repos_url": "https://api.github.com/users/shurcooL/repos",
          "events_url": "https://api.github.com/users/shurcooL/events{/privacy}",
          "received_events_url": "https://api.github.com/users/shurcooL/received_events",
          "type": "User",
          "site_admin": false
        },
        "body": "It would be nice to document ` + "`" + `dir` + "`" + ` and ` + "`" + `path` + "`" + `. How are they used? Why are there 2 parameters instead of just one (` + "`" + `path` + "`" + `)? It's not very obvious to me based on the interface description alone.",
        "created_at": "2016-11-13T23:24:03Z",
        "updated_at": "2016-11-13T23:24:03Z",
        "html_url": "https://github.com/campoy/embedmd/pull/28#discussion_r87725064",
        "pull_request_url": "https://api.github.com/repos/campoy/embedmd/pulls/28",
        "_links": {
          "self": {
            "href": "https://api.github.com/repos/campoy/embedmd/pulls/comments/87725064"
          },
          "html": {
            "href": "https://github.com/campoy/embedmd/pull/28#discussion_r87725064"
          },
          "pull_request": {
            "href": "https://api.github.com/repos/campoy/embedmd/pulls/28"
          }
        }
      },
      "pull_request": {
        "url": "https://api.github.com/repos/campoy/embedmd/pulls/28",
        "id": 93463783,
        "html_url": "https://github.com/campoy/embedmd/pull/28",
        "diff_url": "https://github.com/campoy/embedmd/pull/28.diff",
        "patch_url": "https://github.com/campoy/embedmd/pull/28.patch",
        "issue_url": "https://api.github.com/repos/campoy/embedmd/issues/28",
        "number": 28,
        "state": "open",
        "locked": false,
        "title": "extracting the main functionality into a resuable lib",
        "user": {
          "login": "campoy",
          "id": 2237452,
          "avatar_url": "https://avatars.githubusercontent.com/u/2237452?v=3",
          "gravatar_id": "",
          "url": "https://api.github.com/users/campoy",
          "html_url": "https://github.com/campoy",
          "followers_url": "https://api.github.com/users/campoy/followers",
          "following_url": "https://api.github.com/users/campoy/following{/other_user}",
          "gists_url": "https://api.github.com/users/campoy/gists{/gist_id}",
          "starred_url": "https://api.github.com/users/campoy/starred{/owner}{/repo}",
          "subscriptions_url": "https://api.github.com/users/campoy/subscriptions",
          "organizations_url": "https://api.github.com/users/campoy/orgs",
          "repos_url": "https://api.github.com/users/campoy/repos",
          "events_url": "https://api.github.com/users/campoy/events{/privacy}",
          "received_events_url": "https://api.github.com/users/campoy/received_events",
          "type": "User",
          "site_admin": false
        },
        "body": "Fixes #10.",
        "created_at": "2016-11-13T06:15:16Z",
        "updated_at": "2016-11-13T23:25:01Z",
        "closed_at": null,
        "merged_at": null,
        "merge_commit_sha": "890976751a6f736fc28d3a0a5c892f8d492a38bd",
        "assignee": null,
        "assignees": [

        ],
        "milestone": null,
        "commits_url": "https://api.github.com/repos/campoy/embedmd/pulls/28/commits",
        "review_comments_url": "https://api.github.com/repos/campoy/embedmd/pulls/28/comments",
        "review_comment_url": "https://api.github.com/repos/campoy/embedmd/pulls/comments{/number}",
        "comments_url": "https://api.github.com/repos/campoy/embedmd/issues/28/comments",
        "statuses_url": "https://api.github.com/repos/campoy/embedmd/statuses/7e3080602f24bb83fcd0ce194538f1e1cedb7cfc",
        "head": {
          "label": "campoy:lib",
          "ref": "lib",
          "sha": "7e3080602f24bb83fcd0ce194538f1e1cedb7cfc",
          "user": {
            "login": "campoy",
            "id": 2237452,
            "avatar_url": "https://avatars.githubusercontent.com/u/2237452?v=3",
            "gravatar_id": "",
            "url": "https://api.github.com/users/campoy",
            "html_url": "https://github.com/campoy",
            "followers_url": "https://api.github.com/users/campoy/followers",
            "following_url": "https://api.github.com/users/campoy/following{/other_user}",
            "gists_url": "https://api.github.com/users/campoy/gists{/gist_id}",
            "starred_url": "https://api.github.com/users/campoy/starred{/owner}{/repo}",
            "subscriptions_url": "https://api.github.com/users/campoy/subscriptions",
            "organizations_url": "https://api.github.com/users/campoy/orgs",
            "repos_url": "https://api.github.com/users/campoy/repos",
            "events_url": "https://api.github.com/users/campoy/events{/privacy}",
            "received_events_url": "https://api.github.com/users/campoy/received_events",
            "type": "User",
            "site_admin": false
          },
          "repo": {
            "id": 62099451,
            "name": "embedmd",
            "full_name": "campoy/embedmd",
            "owner": {
              "login": "campoy",
              "id": 2237452,
              "avatar_url": "https://avatars.githubusercontent.com/u/2237452?v=3",
              "gravatar_id": "",
              "url": "https://api.github.com/users/campoy",
              "html_url": "https://github.com/campoy",
              "followers_url": "https://api.github.com/users/campoy/followers",
              "following_url": "https://api.github.com/users/campoy/following{/other_user}",
              "gists_url": "https://api.github.com/users/campoy/gists{/gist_id}",
              "starred_url": "https://api.github.com/users/campoy/starred{/owner}{/repo}",
              "subscriptions_url": "https://api.github.com/users/campoy/subscriptions",
              "organizations_url": "https://api.github.com/users/campoy/orgs",
              "repos_url": "https://api.github.com/users/campoy/repos",
              "events_url": "https://api.github.com/users/campoy/events{/privacy}",
              "received_events_url": "https://api.github.com/users/campoy/received_events",
              "type": "User",
              "site_admin": false
            },
            "private": false,
            "html_url": "https://github.com/campoy/embedmd",
            "description": "embedmd: embed code into markdown and keep everything in sync",
            "fork": false,
            "url": "https://api.github.com/repos/campoy/embedmd",
            "forks_url": "https://api.github.com/repos/campoy/embedmd/forks",
            "keys_url": "https://api.github.com/repos/campoy/embedmd/keys{/key_id}",
            "collaborators_url": "https://api.github.com/repos/campoy/embedmd/collaborators{/collaborator}",
            "teams_url": "https://api.github.com/repos/campoy/embedmd/teams",
            "hooks_url": "https://api.github.com/repos/campoy/embedmd/hooks",
            "issue_events_url": "https://api.github.com/repos/campoy/embedmd/issues/events{/number}",
            "events_url": "https://api.github.com/repos/campoy/embedmd/events",
            "assignees_url": "https://api.github.com/repos/campoy/embedmd/assignees{/user}",
            "branches_url": "https://api.github.com/repos/campoy/embedmd/branches{/branch}",
            "tags_url": "https://api.github.com/repos/campoy/embedmd/tags",
            "blobs_url": "https://api.github.com/repos/campoy/embedmd/git/blobs{/sha}",
            "git_tags_url": "https://api.github.com/repos/campoy/embedmd/git/tags{/sha}",
            "git_refs_url": "https://api.github.com/repos/campoy/embedmd/git/refs{/sha}",
            "trees_url": "https://api.github.com/repos/campoy/embedmd/git/trees{/sha}",
            "statuses_url": "https://api.github.com/repos/campoy/embedmd/statuses/{sha}",
            "languages_url": "https://api.github.com/repos/campoy/embedmd/languages",
            "stargazers_url": "https://api.github.com/repos/campoy/embedmd/stargazers",
            "contributors_url": "https://api.github.com/repos/campoy/embedmd/contributors",
            "subscribers_url": "https://api.github.com/repos/campoy/embedmd/subscribers",
            "subscription_url": "https://api.github.com/repos/campoy/embedmd/subscription",
            "commits_url": "https://api.github.com/repos/campoy/embedmd/commits{/sha}",
            "git_commits_url": "https://api.github.com/repos/campoy/embedmd/git/commits{/sha}",
            "comments_url": "https://api.github.com/repos/campoy/embedmd/comments{/number}",
            "issue_comment_url": "https://api.github.com/repos/campoy/embedmd/issues/comments{/number}",
            "contents_url": "https://api.github.com/repos/campoy/embedmd/contents/{+path}",
            "compare_url": "https://api.github.com/repos/campoy/embedmd/compare/{base}...{head}",
            "merges_url": "https://api.github.com/repos/campoy/embedmd/merges",
            "archive_url": "https://api.github.com/repos/campoy/embedmd/{archive_format}{/ref}",
            "downloads_url": "https://api.github.com/repos/campoy/embedmd/downloads",
            "issues_url": "https://api.github.com/repos/campoy/embedmd/issues{/number}",
            "pulls_url": "https://api.github.com/repos/campoy/embedmd/pulls{/number}",
            "milestones_url": "https://api.github.com/repos/campoy/embedmd/milestones{/number}",
            "notifications_url": "https://api.github.com/repos/campoy/embedmd/notifications{?since,all,participating}",
            "labels_url": "https://api.github.com/repos/campoy/embedmd/labels{/name}",
            "releases_url": "https://api.github.com/repos/campoy/embedmd/releases{/id}",
            "deployments_url": "https://api.github.com/repos/campoy/embedmd/deployments",
            "created_at": "2016-06-28T01:16:46Z",
            "updated_at": "2016-11-13T12:31:02Z",
            "pushed_at": "2016-11-13T08:23:07Z",
            "git_url": "git://github.com/campoy/embedmd.git",
            "ssh_url": "git@github.com:campoy/embedmd.git",
            "clone_url": "https://github.com/campoy/embedmd.git",
            "svn_url": "https://github.com/campoy/embedmd",
            "homepage": "",
            "size": 76,
            "stargazers_count": 293,
            "watchers_count": 293,
            "language": "Go",
            "has_issues": true,
            "has_downloads": true,
            "has_wiki": true,
            "has_pages": false,
            "forks_count": 9,
            "mirror_url": null,
            "open_issues_count": 3,
            "forks": 9,
            "open_issues": 3,
            "watchers": 293,
            "default_branch": "master"
          }
        },
        "base": {
          "label": "campoy:master",
          "ref": "master",
          "sha": "c005cb67a74ca57cf27d2db719c3e244cb440548",
          "user": {
            "login": "campoy",
            "id": 2237452,
            "avatar_url": "https://avatars.githubusercontent.com/u/2237452?v=3",
            "gravatar_id": "",
            "url": "https://api.github.com/users/campoy",
            "html_url": "https://github.com/campoy",
            "followers_url": "https://api.github.com/users/campoy/followers",
            "following_url": "https://api.github.com/users/campoy/following{/other_user}",
            "gists_url": "https://api.github.com/users/campoy/gists{/gist_id}",
            "starred_url": "https://api.github.com/users/campoy/starred{/owner}{/repo}",
            "subscriptions_url": "https://api.github.com/users/campoy/subscriptions",
            "organizations_url": "https://api.github.com/users/campoy/orgs",
            "repos_url": "https://api.github.com/users/campoy/repos",
            "events_url": "https://api.github.com/users/campoy/events{/privacy}",
            "received_events_url": "https://api.github.com/users/campoy/received_events",
            "type": "User",
            "site_admin": false
          },
          "repo": {
            "id": 62099451,
            "name": "embedmd",
            "full_name": "campoy/embedmd",
            "owner": {
              "login": "campoy",
              "id": 2237452,
              "avatar_url": "https://avatars.githubusercontent.com/u/2237452?v=3",
              "gravatar_id": "",
              "url": "https://api.github.com/users/campoy",
              "html_url": "https://github.com/campoy",
              "followers_url": "https://api.github.com/users/campoy/followers",
              "following_url": "https://api.github.com/users/campoy/following{/other_user}",
              "gists_url": "https://api.github.com/users/campoy/gists{/gist_id}",
              "starred_url": "https://api.github.com/users/campoy/starred{/owner}{/repo}",
              "subscriptions_url": "https://api.github.com/users/campoy/subscriptions",
              "organizations_url": "https://api.github.com/users/campoy/orgs",
              "repos_url": "https://api.github.com/users/campoy/repos",
              "events_url": "https://api.github.com/users/campoy/events{/privacy}",
              "received_events_url": "https://api.github.com/users/campoy/received_events",
              "type": "User",
              "site_admin": false
            },
            "private": false,
            "html_url": "https://github.com/campoy/embedmd",
            "description": "embedmd: embed code into markdown and keep everything in sync",
            "fork": false,
            "url": "https://api.github.com/repos/campoy/embedmd",
            "forks_url": "https://api.github.com/repos/campoy/embedmd/forks",
            "keys_url": "https://api.github.com/repos/campoy/embedmd/keys{/key_id}",
            "collaborators_url": "https://api.github.com/repos/campoy/embedmd/collaborators{/collaborator}",
            "teams_url": "https://api.github.com/repos/campoy/embedmd/teams",
            "hooks_url": "https://api.github.com/repos/campoy/embedmd/hooks",
            "issue_events_url": "https://api.github.com/repos/campoy/embedmd/issues/events{/number}",
            "events_url": "https://api.github.com/repos/campoy/embedmd/events",
            "assignees_url": "https://api.github.com/repos/campoy/embedmd/assignees{/user}",
            "branches_url": "https://api.github.com/repos/campoy/embedmd/branches{/branch}",
            "tags_url": "https://api.github.com/repos/campoy/embedmd/tags",
            "blobs_url": "https://api.github.com/repos/campoy/embedmd/git/blobs{/sha}",
            "git_tags_url": "https://api.github.com/repos/campoy/embedmd/git/tags{/sha}",
            "git_refs_url": "https://api.github.com/repos/campoy/embedmd/git/refs{/sha}",
            "trees_url": "https://api.github.com/repos/campoy/embedmd/git/trees{/sha}",
            "statuses_url": "https://api.github.com/repos/campoy/embedmd/statuses/{sha}",
            "languages_url": "https://api.github.com/repos/campoy/embedmd/languages",
            "stargazers_url": "https://api.github.com/repos/campoy/embedmd/stargazers",
            "contributors_url": "https://api.github.com/repos/campoy/embedmd/contributors",
            "subscribers_url": "https://api.github.com/repos/campoy/embedmd/subscribers",
            "subscription_url": "https://api.github.com/repos/campoy/embedmd/subscription",
            "commits_url": "https://api.github.com/repos/campoy/embedmd/commits{/sha}",
            "git_commits_url": "https://api.github.com/repos/campoy/embedmd/git/commits{/sha}",
            "comments_url": "https://api.github.com/repos/campoy/embedmd/comments{/number}",
            "issue_comment_url": "https://api.github.com/repos/campoy/embedmd/issues/comments{/number}",
            "contents_url": "https://api.github.com/repos/campoy/embedmd/contents/{+path}",
            "compare_url": "https://api.github.com/repos/campoy/embedmd/compare/{base}...{head}",
            "merges_url": "https://api.github.com/repos/campoy/embedmd/merges",
            "archive_url": "https://api.github.com/repos/campoy/embedmd/{archive_format}{/ref}",
            "downloads_url": "https://api.github.com/repos/campoy/embedmd/downloads",
            "issues_url": "https://api.github.com/repos/campoy/embedmd/issues{/number}",
            "pulls_url": "https://api.github.com/repos/campoy/embedmd/pulls{/number}",
            "milestones_url": "https://api.github.com/repos/campoy/embedmd/milestones{/number}",
            "notifications_url": "https://api.github.com/repos/campoy/embedmd/notifications{?since,all,participating}",
            "labels_url": "https://api.github.com/repos/campoy/embedmd/labels{/name}",
            "releases_url": "https://api.github.com/repos/campoy/embedmd/releases{/id}",
            "deployments_url": "https://api.github.com/repos/campoy/embedmd/deployments",
            "created_at": "2016-06-28T01:16:46Z",
            "updated_at": "2016-11-13T12:31:02Z",
            "pushed_at": "2016-11-13T08:23:07Z",
            "git_url": "git://github.com/campoy/embedmd.git",
            "ssh_url": "git@github.com:campoy/embedmd.git",
            "clone_url": "https://github.com/campoy/embedmd.git",
            "svn_url": "https://github.com/campoy/embedmd",
            "homepage": "",
            "size": 76,
            "stargazers_count": 293,
            "watchers_count": 293,
            "language": "Go",
            "has_issues": true,
            "has_downloads": true,
            "has_wiki": true,
            "has_pages": false,
            "forks_count": 9,
            "mirror_url": null,
            "open_issues_count": 3,
            "forks": 9,
            "open_issues": 3,
            "watchers": 293,
            "default_branch": "master"
          }
        },
        "_links": {
          "self": {
            "href": "https://api.github.com/repos/campoy/embedmd/pulls/28"
          },
          "html": {
            "href": "https://github.com/campoy/embedmd/pull/28"
          },
          "issue": {
            "href": "https://api.github.com/repos/campoy/embedmd/issues/28"
          },
          "comments": {
            "href": "https://api.github.com/repos/campoy/embedmd/issues/28/comments"
          },
          "review_comments": {
            "href": "https://api.github.com/repos/campoy/embedmd/pulls/28/comments"
          },
          "review_comment": {
            "href": "https://api.github.com/repos/campoy/embedmd/pulls/comments{/number}"
          },
          "commits": {
            "href": "https://api.github.com/repos/campoy/embedmd/pulls/28/commits"
          },
          "statuses": {
            "href": "https://api.github.com/repos/campoy/embedmd/statuses/7e3080602f24bb83fcd0ce194538f1e1cedb7cfc"
          }
        }
      }
    },
    "public": true,
    "created_at": "2016-11-13T23:24:03Z"
  },
  {
    "id": "4860085536",
    "type": "PullRequestReviewCommentEvent",
    "actor": {
      "id": 1924134,
      "login": "shurcooL",
      "display_login": "shurcooL",
      "gravatar_id": "",
      "url": "https://api.github.com/users/shurcooL",
      "avatar_url": "https://avatars.githubusercontent.com/u/1924134?"
    },
    "repo": {
      "id": 62099451,
      "name": "campoy/embedmd",
      "url": "https://api.github.com/repos/campoy/embedmd"
    },
    "payload": {
      "action": "created",
      "comment": {
        "url": "https://api.github.com/repos/campoy/embedmd/pulls/comments/87725032",
        "pull_request_review_id": 8325288,
        "id": 87725032,
        "diff_hunk": "@@ -0,0 +1,48 @@\n+// Copyright 2016 Google Inc. All rights reserved.\n+// Licensed under the Apache License, Version 2.0 (the \"License\");\n+// you may not use this file except in compliance with the License.\n+// You may obtain a copy of the License at\n+// http://www.apache.org/licenses/LICENSE-2.0\n+//\n+// Unless required by applicable law or agreed to writing, software distributed\n+// under the License is distributed on a \"AS IS\" BASIS, WITHOUT WARRANTIES OR\n+// CONDITIONS OF ANY KIND, either express or implied.\n+//\n+// See the License for the specific language governing permissions and\n+// limitations under the License.\n+\n+package embedmd\n+\n+import (\n+\t\"fmt\"\n+\t\"io/ioutil\"\n+\t\"net/http\"\n+\t\"path/filepath\"\n+\t\"strings\"\n+)\n+\n+// Fetcher provides an abstraction on a file system.\n+// The Fetch function is called anytime some content needs to be fetched.\n+// For now this includes files and URLs.\n+type Fetcher interface {\n+\tFetch(dir, path string) ([]byte, error)\n+}\n+\n+type fetcher struct{}\n+\n+func (fetcher) Fetch(dir, path string) ([]byte, error) {\n+\tif !strings.HasPrefix(path, \"http://\") && !strings.HasPrefix(path, \"https://\") {\n+\t\tpath = filepath.Join(filepath.FromSlash(path))",
        "path": "embedmd/content.go",
        "position": 35,
        "original_position": 35,
        "commit_id": "7e3080602f24bb83fcd0ce194538f1e1cedb7cfc",
        "original_commit_id": "7e3080602f24bb83fcd0ce194538f1e1cedb7cfc",
        "user": {
          "login": "shurcooL",
          "id": 1924134,
          "avatar_url": "https://avatars.githubusercontent.com/u/1924134?v=3",
          "gravatar_id": "",
          "url": "https://api.github.com/users/shurcooL",
          "html_url": "https://github.com/shurcooL",
          "followers_url": "https://api.github.com/users/shurcooL/followers",
          "following_url": "https://api.github.com/users/shurcooL/following{/other_user}",
          "gists_url": "https://api.github.com/users/shurcooL/gists{/gist_id}",
          "starred_url": "https://api.github.com/users/shurcooL/starred{/owner}{/repo}",
          "subscriptions_url": "https://api.github.com/users/shurcooL/subscriptions",
          "organizations_url": "https://api.github.com/users/shurcooL/orgs",
          "repos_url": "https://api.github.com/users/shurcooL/repos",
          "events_url": "https://api.github.com/users/shurcooL/events{/privacy}",
          "received_events_url": "https://api.github.com/users/shurcooL/received_events",
          "type": "User",
          "site_admin": false
        },
        "body": "Are you missing ` + "`" + `dir` + "`" + ` in this line? ` + "`" + `filepath.Join` + "`" + ` has only one element, and ` + "`" + `dir` + "`" + ` is completely unused.\r\n\r\nAlso, ` + "`" + `mixedContentProvider.Fetch` + "`" + `, which is very similar, does ` + "`" + `filepath.Join(dir, filepath.FromSlash(path))` + "`" + `.",
        "created_at": "2016-11-13T23:23:12Z",
        "updated_at": "2016-11-13T23:24:53Z",
        "html_url": "https://github.com/campoy/embedmd/pull/28#discussion_r87725032",
        "pull_request_url": "https://api.github.com/repos/campoy/embedmd/pulls/28",
        "_links": {
          "self": {
            "href": "https://api.github.com/repos/campoy/embedmd/pulls/comments/87725032"
          },
          "html": {
            "href": "https://github.com/campoy/embedmd/pull/28#discussion_r87725032"
          },
          "pull_request": {
            "href": "https://api.github.com/repos/campoy/embedmd/pulls/28"
          }
        }
      },
      "pull_request": {
        "url": "https://api.github.com/repos/campoy/embedmd/pulls/28",
        "id": 93463783,
        "html_url": "https://github.com/campoy/embedmd/pull/28",
        "diff_url": "https://github.com/campoy/embedmd/pull/28.diff",
        "patch_url": "https://github.com/campoy/embedmd/pull/28.patch",
        "issue_url": "https://api.github.com/repos/campoy/embedmd/issues/28",
        "number": 28,
        "state": "open",
        "locked": false,
        "title": "extracting the main functionality into a resuable lib",
        "user": {
          "login": "campoy",
          "id": 2237452,
          "avatar_url": "https://avatars.githubusercontent.com/u/2237452?v=3",
          "gravatar_id": "",
          "url": "https://api.github.com/users/campoy",
          "html_url": "https://github.com/campoy",
          "followers_url": "https://api.github.com/users/campoy/followers",
          "following_url": "https://api.github.com/users/campoy/following{/other_user}",
          "gists_url": "https://api.github.com/users/campoy/gists{/gist_id}",
          "starred_url": "https://api.github.com/users/campoy/starred{/owner}{/repo}",
          "subscriptions_url": "https://api.github.com/users/campoy/subscriptions",
          "organizations_url": "https://api.github.com/users/campoy/orgs",
          "repos_url": "https://api.github.com/users/campoy/repos",
          "events_url": "https://api.github.com/users/campoy/events{/privacy}",
          "received_events_url": "https://api.github.com/users/campoy/received_events",
          "type": "User",
          "site_admin": false
        },
        "body": "Fixes #10.",
        "created_at": "2016-11-13T06:15:16Z",
        "updated_at": "2016-11-13T23:24:53Z",
        "closed_at": null,
        "merged_at": null,
        "merge_commit_sha": "890976751a6f736fc28d3a0a5c892f8d492a38bd",
        "assignee": null,
        "assignees": [

        ],
        "milestone": null,
        "commits_url": "https://api.github.com/repos/campoy/embedmd/pulls/28/commits",
        "review_comments_url": "https://api.github.com/repos/campoy/embedmd/pulls/28/comments",
        "review_comment_url": "https://api.github.com/repos/campoy/embedmd/pulls/comments{/number}",
        "comments_url": "https://api.github.com/repos/campoy/embedmd/issues/28/comments",
        "statuses_url": "https://api.github.com/repos/campoy/embedmd/statuses/7e3080602f24bb83fcd0ce194538f1e1cedb7cfc",
        "head": {
          "label": "campoy:lib",
          "ref": "lib",
          "sha": "7e3080602f24bb83fcd0ce194538f1e1cedb7cfc",
          "user": {
            "login": "campoy",
            "id": 2237452,
            "avatar_url": "https://avatars.githubusercontent.com/u/2237452?v=3",
            "gravatar_id": "",
            "url": "https://api.github.com/users/campoy",
            "html_url": "https://github.com/campoy",
            "followers_url": "https://api.github.com/users/campoy/followers",
            "following_url": "https://api.github.com/users/campoy/following{/other_user}",
            "gists_url": "https://api.github.com/users/campoy/gists{/gist_id}",
            "starred_url": "https://api.github.com/users/campoy/starred{/owner}{/repo}",
            "subscriptions_url": "https://api.github.com/users/campoy/subscriptions",
            "organizations_url": "https://api.github.com/users/campoy/orgs",
            "repos_url": "https://api.github.com/users/campoy/repos",
            "events_url": "https://api.github.com/users/campoy/events{/privacy}",
            "received_events_url": "https://api.github.com/users/campoy/received_events",
            "type": "User",
            "site_admin": false
          },
          "repo": {
            "id": 62099451,
            "name": "embedmd",
            "full_name": "campoy/embedmd",
            "owner": {
              "login": "campoy",
              "id": 2237452,
              "avatar_url": "https://avatars.githubusercontent.com/u/2237452?v=3",
              "gravatar_id": "",
              "url": "https://api.github.com/users/campoy",
              "html_url": "https://github.com/campoy",
              "followers_url": "https://api.github.com/users/campoy/followers",
              "following_url": "https://api.github.com/users/campoy/following{/other_user}",
              "gists_url": "https://api.github.com/users/campoy/gists{/gist_id}",
              "starred_url": "https://api.github.com/users/campoy/starred{/owner}{/repo}",
              "subscriptions_url": "https://api.github.com/users/campoy/subscriptions",
              "organizations_url": "https://api.github.com/users/campoy/orgs",
              "repos_url": "https://api.github.com/users/campoy/repos",
              "events_url": "https://api.github.com/users/campoy/events{/privacy}",
              "received_events_url": "https://api.github.com/users/campoy/received_events",
              "type": "User",
              "site_admin": false
            },
            "private": false,
            "html_url": "https://github.com/campoy/embedmd",
            "description": "embedmd: embed code into markdown and keep everything in sync",
            "fork": false,
            "url": "https://api.github.com/repos/campoy/embedmd",
            "forks_url": "https://api.github.com/repos/campoy/embedmd/forks",
            "keys_url": "https://api.github.com/repos/campoy/embedmd/keys{/key_id}",
            "collaborators_url": "https://api.github.com/repos/campoy/embedmd/collaborators{/collaborator}",
            "teams_url": "https://api.github.com/repos/campoy/embedmd/teams",
            "hooks_url": "https://api.github.com/repos/campoy/embedmd/hooks",
            "issue_events_url": "https://api.github.com/repos/campoy/embedmd/issues/events{/number}",
            "events_url": "https://api.github.com/repos/campoy/embedmd/events",
            "assignees_url": "https://api.github.com/repos/campoy/embedmd/assignees{/user}",
            "branches_url": "https://api.github.com/repos/campoy/embedmd/branches{/branch}",
            "tags_url": "https://api.github.com/repos/campoy/embedmd/tags",
            "blobs_url": "https://api.github.com/repos/campoy/embedmd/git/blobs{/sha}",
            "git_tags_url": "https://api.github.com/repos/campoy/embedmd/git/tags{/sha}",
            "git_refs_url": "https://api.github.com/repos/campoy/embedmd/git/refs{/sha}",
            "trees_url": "https://api.github.com/repos/campoy/embedmd/git/trees{/sha}",
            "statuses_url": "https://api.github.com/repos/campoy/embedmd/statuses/{sha}",
            "languages_url": "https://api.github.com/repos/campoy/embedmd/languages",
            "stargazers_url": "https://api.github.com/repos/campoy/embedmd/stargazers",
            "contributors_url": "https://api.github.com/repos/campoy/embedmd/contributors",
            "subscribers_url": "https://api.github.com/repos/campoy/embedmd/subscribers",
            "subscription_url": "https://api.github.com/repos/campoy/embedmd/subscription",
            "commits_url": "https://api.github.com/repos/campoy/embedmd/commits{/sha}",
            "git_commits_url": "https://api.github.com/repos/campoy/embedmd/git/commits{/sha}",
            "comments_url": "https://api.github.com/repos/campoy/embedmd/comments{/number}",
            "issue_comment_url": "https://api.github.com/repos/campoy/embedmd/issues/comments{/number}",
            "contents_url": "https://api.github.com/repos/campoy/embedmd/contents/{+path}",
            "compare_url": "https://api.github.com/repos/campoy/embedmd/compare/{base}...{head}",
            "merges_url": "https://api.github.com/repos/campoy/embedmd/merges",
            "archive_url": "https://api.github.com/repos/campoy/embedmd/{archive_format}{/ref}",
            "downloads_url": "https://api.github.com/repos/campoy/embedmd/downloads",
            "issues_url": "https://api.github.com/repos/campoy/embedmd/issues{/number}",
            "pulls_url": "https://api.github.com/repos/campoy/embedmd/pulls{/number}",
            "milestones_url": "https://api.github.com/repos/campoy/embedmd/milestones{/number}",
            "notifications_url": "https://api.github.com/repos/campoy/embedmd/notifications{?since,all,participating}",
            "labels_url": "https://api.github.com/repos/campoy/embedmd/labels{/name}",
            "releases_url": "https://api.github.com/repos/campoy/embedmd/releases{/id}",
            "deployments_url": "https://api.github.com/repos/campoy/embedmd/deployments",
            "created_at": "2016-06-28T01:16:46Z",
            "updated_at": "2016-11-13T12:31:02Z",
            "pushed_at": "2016-11-13T08:23:07Z",
            "git_url": "git://github.com/campoy/embedmd.git",
            "ssh_url": "git@github.com:campoy/embedmd.git",
            "clone_url": "https://github.com/campoy/embedmd.git",
            "svn_url": "https://github.com/campoy/embedmd",
            "homepage": "",
            "size": 76,
            "stargazers_count": 293,
            "watchers_count": 293,
            "language": "Go",
            "has_issues": true,
            "has_downloads": true,
            "has_wiki": true,
            "has_pages": false,
            "forks_count": 9,
            "mirror_url": null,
            "open_issues_count": 3,
            "forks": 9,
            "open_issues": 3,
            "watchers": 293,
            "default_branch": "master"
          }
        },
        "base": {
          "label": "campoy:master",
          "ref": "master",
          "sha": "c005cb67a74ca57cf27d2db719c3e244cb440548",
          "user": {
            "login": "campoy",
            "id": 2237452,
            "avatar_url": "https://avatars.githubusercontent.com/u/2237452?v=3",
            "gravatar_id": "",
            "url": "https://api.github.com/users/campoy",
            "html_url": "https://github.com/campoy",
            "followers_url": "https://api.github.com/users/campoy/followers",
            "following_url": "https://api.github.com/users/campoy/following{/other_user}",
            "gists_url": "https://api.github.com/users/campoy/gists{/gist_id}",
            "starred_url": "https://api.github.com/users/campoy/starred{/owner}{/repo}",
            "subscriptions_url": "https://api.github.com/users/campoy/subscriptions",
            "organizations_url": "https://api.github.com/users/campoy/orgs",
            "repos_url": "https://api.github.com/users/campoy/repos",
            "events_url": "https://api.github.com/users/campoy/events{/privacy}",
            "received_events_url": "https://api.github.com/users/campoy/received_events",
            "type": "User",
            "site_admin": false
          },
          "repo": {
            "id": 62099451,
            "name": "embedmd",
            "full_name": "campoy/embedmd",
            "owner": {
              "login": "campoy",
              "id": 2237452,
              "avatar_url": "https://avatars.githubusercontent.com/u/2237452?v=3",
              "gravatar_id": "",
              "url": "https://api.github.com/users/campoy",
              "html_url": "https://github.com/campoy",
              "followers_url": "https://api.github.com/users/campoy/followers",
              "following_url": "https://api.github.com/users/campoy/following{/other_user}",
              "gists_url": "https://api.github.com/users/campoy/gists{/gist_id}",
              "starred_url": "https://api.github.com/users/campoy/starred{/owner}{/repo}",
              "subscriptions_url": "https://api.github.com/users/campoy/subscriptions",
              "organizations_url": "https://api.github.com/users/campoy/orgs",
              "repos_url": "https://api.github.com/users/campoy/repos",
              "events_url": "https://api.github.com/users/campoy/events{/privacy}",
              "received_events_url": "https://api.github.com/users/campoy/received_events",
              "type": "User",
              "site_admin": false
            },
            "private": false,
            "html_url": "https://github.com/campoy/embedmd",
            "description": "embedmd: embed code into markdown and keep everything in sync",
            "fork": false,
            "url": "https://api.github.com/repos/campoy/embedmd",
            "forks_url": "https://api.github.com/repos/campoy/embedmd/forks",
            "keys_url": "https://api.github.com/repos/campoy/embedmd/keys{/key_id}",
            "collaborators_url": "https://api.github.com/repos/campoy/embedmd/collaborators{/collaborator}",
            "teams_url": "https://api.github.com/repos/campoy/embedmd/teams",
            "hooks_url": "https://api.github.com/repos/campoy/embedmd/hooks",
            "issue_events_url": "https://api.github.com/repos/campoy/embedmd/issues/events{/number}",
            "events_url": "https://api.github.com/repos/campoy/embedmd/events",
            "assignees_url": "https://api.github.com/repos/campoy/embedmd/assignees{/user}",
            "branches_url": "https://api.github.com/repos/campoy/embedmd/branches{/branch}",
            "tags_url": "https://api.github.com/repos/campoy/embedmd/tags",
            "blobs_url": "https://api.github.com/repos/campoy/embedmd/git/blobs{/sha}",
            "git_tags_url": "https://api.github.com/repos/campoy/embedmd/git/tags{/sha}",
            "git_refs_url": "https://api.github.com/repos/campoy/embedmd/git/refs{/sha}",
            "trees_url": "https://api.github.com/repos/campoy/embedmd/git/trees{/sha}",
            "statuses_url": "https://api.github.com/repos/campoy/embedmd/statuses/{sha}",
            "languages_url": "https://api.github.com/repos/campoy/embedmd/languages",
            "stargazers_url": "https://api.github.com/repos/campoy/embedmd/stargazers",
            "contributors_url": "https://api.github.com/repos/campoy/embedmd/contributors",
            "subscribers_url": "https://api.github.com/repos/campoy/embedmd/subscribers",
            "subscription_url": "https://api.github.com/repos/campoy/embedmd/subscription",
            "commits_url": "https://api.github.com/repos/campoy/embedmd/commits{/sha}",
            "git_commits_url": "https://api.github.com/repos/campoy/embedmd/git/commits{/sha}",
            "comments_url": "https://api.github.com/repos/campoy/embedmd/comments{/number}",
            "issue_comment_url": "https://api.github.com/repos/campoy/embedmd/issues/comments{/number}",
            "contents_url": "https://api.github.com/repos/campoy/embedmd/contents/{+path}",
            "compare_url": "https://api.github.com/repos/campoy/embedmd/compare/{base}...{head}",
            "merges_url": "https://api.github.com/repos/campoy/embedmd/merges",
            "archive_url": "https://api.github.com/repos/campoy/embedmd/{archive_format}{/ref}",
            "downloads_url": "https://api.github.com/repos/campoy/embedmd/downloads",
            "issues_url": "https://api.github.com/repos/campoy/embedmd/issues{/number}",
            "pulls_url": "https://api.github.com/repos/campoy/embedmd/pulls{/number}",
            "milestones_url": "https://api.github.com/repos/campoy/embedmd/milestones{/number}",
            "notifications_url": "https://api.github.com/repos/campoy/embedmd/notifications{?since,all,participating}",
            "labels_url": "https://api.github.com/repos/campoy/embedmd/labels{/name}",
            "releases_url": "https://api.github.com/repos/campoy/embedmd/releases{/id}",
            "deployments_url": "https://api.github.com/repos/campoy/embedmd/deployments",
            "created_at": "2016-06-28T01:16:46Z",
            "updated_at": "2016-11-13T12:31:02Z",
            "pushed_at": "2016-11-13T08:23:07Z",
            "git_url": "git://github.com/campoy/embedmd.git",
            "ssh_url": "git@github.com:campoy/embedmd.git",
            "clone_url": "https://github.com/campoy/embedmd.git",
            "svn_url": "https://github.com/campoy/embedmd",
            "homepage": "",
            "size": 76,
            "stargazers_count": 293,
            "watchers_count": 293,
            "language": "Go",
            "has_issues": true,
            "has_downloads": true,
            "has_wiki": true,
            "has_pages": false,
            "forks_count": 9,
            "mirror_url": null,
            "open_issues_count": 3,
            "forks": 9,
            "open_issues": 3,
            "watchers": 293,
            "default_branch": "master"
          }
        },
        "_links": {
          "self": {
            "href": "https://api.github.com/repos/campoy/embedmd/pulls/28"
          },
          "html": {
            "href": "https://github.com/campoy/embedmd/pull/28"
          },
          "issue": {
            "href": "https://api.github.com/repos/campoy/embedmd/issues/28"
          },
          "comments": {
            "href": "https://api.github.com/repos/campoy/embedmd/issues/28/comments"
          },
          "review_comments": {
            "href": "https://api.github.com/repos/campoy/embedmd/pulls/28/comments"
          },
          "review_comment": {
            "href": "https://api.github.com/repos/campoy/embedmd/pulls/comments{/number}"
          },
          "commits": {
            "href": "https://api.github.com/repos/campoy/embedmd/pulls/28/commits"
          },
          "statuses": {
            "href": "https://api.github.com/repos/campoy/embedmd/statuses/7e3080602f24bb83fcd0ce194538f1e1cedb7cfc"
          }
        }
      }
    },
    "public": true,
    "created_at": "2016-11-13T23:23:12Z"
  },
  {
    "id": "4857967959",
    "type": "IssueCommentEvent",
    "actor": {
      "id": 1924134,
      "login": "shurcooL",
      "display_login": "shurcooL",
      "gravatar_id": "",
      "url": "https://api.github.com/users/shurcooL",
      "avatar_url": "https://avatars.githubusercontent.com/u/1924134?"
    },
    "repo": {
      "id": 26097793,
      "name": "gopherjs/vecty",
      "url": "https://api.github.com/repos/gopherjs/vecty"
    },
    "payload": {
      "action": "created",
      "issue": {
        "url": "https://api.github.com/repos/gopherjs/vecty/issues/65",
        "repository_url": "https://api.github.com/repos/gopherjs/vecty",
        "labels_url": "https://api.github.com/repos/gopherjs/vecty/issues/65/labels{/name}",
        "comments_url": "https://api.github.com/repos/gopherjs/vecty/issues/65/comments",
        "events_url": "https://api.github.com/repos/gopherjs/vecty/issues/65/events",
        "html_url": "https://github.com/gopherjs/vecty/issues/65",
        "id": 188914921,
        "number": 65,
        "title": "Need to expose the native event object",
        "user": {
          "login": "davelondon",
          "id": 925351,
          "avatar_url": "https://avatars.githubusercontent.com/u/925351?v=3",
          "gravatar_id": "",
          "url": "https://api.github.com/users/davelondon",
          "html_url": "https://github.com/davelondon",
          "followers_url": "https://api.github.com/users/davelondon/followers",
          "following_url": "https://api.github.com/users/davelondon/following{/other_user}",
          "gists_url": "https://api.github.com/users/davelondon/gists{/gist_id}",
          "starred_url": "https://api.github.com/users/davelondon/starred{/owner}{/repo}",
          "subscriptions_url": "https://api.github.com/users/davelondon/subscriptions",
          "organizations_url": "https://api.github.com/users/davelondon/orgs",
          "repos_url": "https://api.github.com/users/davelondon/repos",
          "events_url": "https://api.github.com/users/davelondon/events{/privacy}",
          "received_events_url": "https://api.github.com/users/davelondon/received_events",
          "type": "User",
          "site_admin": false
        },
        "labels": [

        ],
        "state": "open",
        "locked": false,
        "assignee": null,
        "assignees": [

        ],
        "milestone": null,
        "comments": 1,
        "created_at": "2016-11-12T14:40:59Z",
        "updated_at": "2016-11-12T21:55:52Z",
        "closed_at": null,
        "body": "In order to do a drag+drop file upload, you need access to ` + "`" + `dataTransfer` + "`" + ` from the js event. The js native event object isn't currently presented to the vecty event handler. I've added this in my fork as an anonymous struct member, and it seems to feel good in use:\r\n\r\nhttps://github.com/davelondon/uploader/blob/master/editor.go#L133\r\n\r\nI seem to remember this was discussed before?"
      },
      "comment": {
        "url": "https://api.github.com/repos/gopherjs/vecty/issues/comments/260151016",
        "html_url": "https://github.com/gopherjs/vecty/issues/65#issuecomment-260151016",
        "issue_url": "https://api.github.com/repos/gopherjs/vecty/issues/65",
        "id": 260151016,
        "user": {
          "login": "shurcooL",
          "id": 1924134,
          "avatar_url": "https://avatars.githubusercontent.com/u/1924134?v=3",
          "gravatar_id": "",
          "url": "https://api.github.com/users/shurcooL",
          "html_url": "https://github.com/shurcooL",
          "followers_url": "https://api.github.com/users/shurcooL/followers",
          "following_url": "https://api.github.com/users/shurcooL/following{/other_user}",
          "gists_url": "https://api.github.com/users/shurcooL/gists{/gist_id}",
          "starred_url": "https://api.github.com/users/shurcooL/starred{/owner}{/repo}",
          "subscriptions_url": "https://api.github.com/users/shurcooL/subscriptions",
          "organizations_url": "https://api.github.com/users/shurcooL/orgs",
          "repos_url": "https://api.github.com/users/shurcooL/repos",
          "events_url": "https://api.github.com/users/shurcooL/events{/privacy}",
          "received_events_url": "https://api.github.com/users/shurcooL/received_events",
          "type": "User",
          "site_admin": false
        },
        "created_at": "2016-11-12T21:55:52Z",
        "updated_at": "2016-11-12T21:55:52Z",
        "body": "> need access to ` + "`" + `dataTransfer` + "`" + ` from the js event. The js native event object isn't currently presented to the vecty event handler.\r\n\r\nJust asking, but have you considered simply filling that gap and exposing ` + "`" + `dataTransfer` + "`" + ` natively?"
      }
    },
    "public": true,
    "created_at": "2016-11-12T21:55:52Z",
    "org": {
      "id": 6654647,
      "login": "gopherjs",
      "gravatar_id": "",
      "url": "https://api.github.com/orgs/gopherjs",
      "avatar_url": "https://avatars.githubusercontent.com/u/6654647?"
    }
  },
  {
    "id": "4857955520",
    "type": "WatchEvent",
    "actor": {
      "id": 1924134,
      "login": "shurcooL",
      "display_login": "shurcooL",
      "gravatar_id": "",
      "url": "https://api.github.com/users/shurcooL",
      "avatar_url": "https://avatars.githubusercontent.com/u/1924134?"
    },
    "repo": {
      "id": 72888736,
      "name": "dop251/goja",
      "url": "https://api.github.com/repos/dop251/goja"
    },
    "payload": {
      "action": "started"
    },
    "public": true,
    "created_at": "2016-11-12T21:47:46Z"
  },
  {
    "id": "4857946860",
    "type": "IssueCommentEvent",
    "actor": {
      "id": 1924134,
      "login": "shurcooL",
      "display_login": "shurcooL",
      "gravatar_id": "",
      "url": "https://api.github.com/users/shurcooL",
      "avatar_url": "https://avatars.githubusercontent.com/u/1924134?"
    },
    "repo": {
      "id": 10270722,
      "name": "google/go-github",
      "url": "https://api.github.com/repos/google/go-github"
    },
    "payload": {
      "action": "created",
      "issue": {
        "url": "https://api.github.com/repos/google/go-github/issues/472",
        "repository_url": "https://api.github.com/repos/google/go-github",
        "labels_url": "https://api.github.com/repos/google/go-github/issues/472/labels{/name}",
        "comments_url": "https://api.github.com/repos/google/go-github/issues/472/comments",
        "events_url": "https://api.github.com/repos/google/go-github/issues/472/events",
        "html_url": "https://github.com/google/go-github/pull/472",
        "id": 188935876,
        "number": 472,
        "title": "Update License struct and add new RepositoryLicense struct",
        "user": {
          "login": "nmiyake",
          "id": 4267425,
          "avatar_url": "https://avatars.githubusercontent.com/u/4267425?v=3",
          "gravatar_id": "",
          "url": "https://api.github.com/users/nmiyake",
          "html_url": "https://github.com/nmiyake",
          "followers_url": "https://api.github.com/users/nmiyake/followers",
          "following_url": "https://api.github.com/users/nmiyake/following{/other_user}",
          "gists_url": "https://api.github.com/users/nmiyake/gists{/gist_id}",
          "starred_url": "https://api.github.com/users/nmiyake/starred{/owner}{/repo}",
          "subscriptions_url": "https://api.github.com/users/nmiyake/subscriptions",
          "organizations_url": "https://api.github.com/users/nmiyake/orgs",
          "repos_url": "https://api.github.com/users/nmiyake/repos",
          "events_url": "https://api.github.com/users/nmiyake/events{/privacy}",
          "received_events_url": "https://api.github.com/users/nmiyake/received_events",
          "type": "User",
          "site_admin": false
        },
        "labels": [

        ],
        "state": "open",
        "locked": false,
        "assignee": null,
        "assignees": [

        ],
        "milestone": null,
        "comments": 1,
        "created_at": "2016-11-12T21:30:43Z",
        "updated_at": "2016-11-12T21:42:22Z",
        "closed_at": null,
        "pull_request": {
          "url": "https://api.github.com/repos/google/go-github/pulls/472",
          "html_url": "https://github.com/google/go-github/pull/472",
          "diff_url": "https://github.com/google/go-github/pull/472.diff",
          "patch_url": "https://github.com/google/go-github/pull/472.patch"
        },
        "body": "Update structs to reflect the new reponses returned from GitHub.\r\n\r\nFixes #471"
      },
      "comment": {
        "url": "https://api.github.com/repos/google/go-github/issues/comments/260150258",
        "html_url": "https://github.com/google/go-github/pull/472#issuecomment-260150258",
        "issue_url": "https://api.github.com/repos/google/go-github/issues/472",
        "id": 260150258,
        "user": {
          "login": "shurcooL",
          "id": 1924134,
          "avatar_url": "https://avatars.githubusercontent.com/u/1924134?v=3",
          "gravatar_id": "",
          "url": "https://api.github.com/users/shurcooL",
          "html_url": "https://github.com/shurcooL",
          "followers_url": "https://api.github.com/users/shurcooL/followers",
          "following_url": "https://api.github.com/users/shurcooL/following{/other_user}",
          "gists_url": "https://api.github.com/users/shurcooL/gists{/gist_id}",
          "starred_url": "https://api.github.com/users/shurcooL/starred{/owner}{/repo}",
          "subscriptions_url": "https://api.github.com/users/shurcooL/subscriptions",
          "organizations_url": "https://api.github.com/users/shurcooL/orgs",
          "repos_url": "https://api.github.com/users/shurcooL/repos",
          "events_url": "https://api.github.com/users/shurcooL/events{/privacy}",
          "received_events_url": "https://api.github.com/users/shurcooL/received_events",
          "type": "User",
          "site_admin": false
        },
        "created_at": "2016-11-12T21:42:22Z",
        "updated_at": "2016-11-12T21:42:22Z",
        "body": "LGTM. Thanks for reporting and fixing this issue @nmiyake!"
      }
    },
    "public": true,
    "created_at": "2016-11-12T21:42:23Z",
    "org": {
      "id": 1342004,
      "login": "google",
      "gravatar_id": "",
      "url": "https://api.github.com/orgs/google",
      "avatar_url": "https://avatars.githubusercontent.com/u/1342004?"
    }
  },
  {
    "id": "4857936603",
    "type": "PullRequestReviewCommentEvent",
    "actor": {
      "id": 1924134,
      "login": "shurcooL",
      "display_login": "shurcooL",
      "gravatar_id": "",
      "url": "https://api.github.com/users/shurcooL",
      "avatar_url": "https://avatars.githubusercontent.com/u/1924134?"
    },
    "repo": {
      "id": 10270722,
      "name": "google/go-github",
      "url": "https://api.github.com/repos/google/go-github"
    },
    "payload": {
      "action": "created",
      "comment": {
        "url": "https://api.github.com/repos/google/go-github/pulls/comments/87700697",
        "pull_request_review_id": 8304983,
        "id": 87700697,
        "diff_hunk": "@@ -10,23 +10,49 @@ import \"fmt\"\n // LicensesService handles communication with the license related\n // methods of the GitHub API.\n //\n-// GitHub API docs: http://developer.github.com/v3/pulls/\n+// GitHub API docs: http://developer.github.com/v3/licenses/",
        "path": "github/licenses.go",
        "position": 5,
        "original_position": 5,
        "commit_id": "d04745e0a3f0bf2eb9d5c01989a672b91bcd9791",
        "original_commit_id": "d04745e0a3f0bf2eb9d5c01989a672b91bcd9791",
        "user": {
          "login": "shurcooL",
          "id": 1924134,
          "avatar_url": "https://avatars.githubusercontent.com/u/1924134?v=3",
          "gravatar_id": "",
          "url": "https://api.github.com/users/shurcooL",
          "html_url": "https://github.com/shurcooL",
          "followers_url": "https://api.github.com/users/shurcooL/followers",
          "following_url": "https://api.github.com/users/shurcooL/following{/other_user}",
          "gists_url": "https://api.github.com/users/shurcooL/gists{/gist_id}",
          "starred_url": "https://api.github.com/users/shurcooL/starred{/owner}{/repo}",
          "subscriptions_url": "https://api.github.com/users/shurcooL/subscriptions",
          "organizations_url": "https://api.github.com/users/shurcooL/orgs",
          "repos_url": "https://api.github.com/users/shurcooL/repos",
          "events_url": "https://api.github.com/users/shurcooL/events{/privacy}",
          "received_events_url": "https://api.github.com/users/shurcooL/received_events",
          "type": "User",
          "site_admin": false
        },
        "body": "Mind changing the scheme to https here?",
        "created_at": "2016-11-12T21:35:46Z",
        "updated_at": "2016-11-12T21:35:46Z",
        "html_url": "https://github.com/google/go-github/pull/472#discussion_r87700697",
        "pull_request_url": "https://api.github.com/repos/google/go-github/pulls/472",
        "_links": {
          "self": {
            "href": "https://api.github.com/repos/google/go-github/pulls/comments/87700697"
          },
          "html": {
            "href": "https://github.com/google/go-github/pull/472#discussion_r87700697"
          },
          "pull_request": {
            "href": "https://api.github.com/repos/google/go-github/pulls/472"
          }
        }
      },
      "pull_request": {
        "url": "https://api.github.com/repos/google/go-github/pulls/472",
        "id": 93452704,
        "html_url": "https://github.com/google/go-github/pull/472",
        "diff_url": "https://github.com/google/go-github/pull/472.diff",
        "patch_url": "https://github.com/google/go-github/pull/472.patch",
        "issue_url": "https://api.github.com/repos/google/go-github/issues/472",
        "number": 472,
        "state": "open",
        "locked": false,
        "title": "Update License struct and add new RepositoryLicense struct",
        "user": {
          "login": "nmiyake",
          "id": 4267425,
          "avatar_url": "https://avatars.githubusercontent.com/u/4267425?v=3",
          "gravatar_id": "",
          "url": "https://api.github.com/users/nmiyake",
          "html_url": "https://github.com/nmiyake",
          "followers_url": "https://api.github.com/users/nmiyake/followers",
          "following_url": "https://api.github.com/users/nmiyake/following{/other_user}",
          "gists_url": "https://api.github.com/users/nmiyake/gists{/gist_id}",
          "starred_url": "https://api.github.com/users/nmiyake/starred{/owner}{/repo}",
          "subscriptions_url": "https://api.github.com/users/nmiyake/subscriptions",
          "organizations_url": "https://api.github.com/users/nmiyake/orgs",
          "repos_url": "https://api.github.com/users/nmiyake/repos",
          "events_url": "https://api.github.com/users/nmiyake/events{/privacy}",
          "received_events_url": "https://api.github.com/users/nmiyake/received_events",
          "type": "User",
          "site_admin": false
        },
        "body": "Update structs to reflect the new reponses returned from GitHub.\r\n\r\nFixes #471",
        "created_at": "2016-11-12T21:30:43Z",
        "updated_at": "2016-11-12T21:35:46Z",
        "closed_at": null,
        "merged_at": null,
        "merge_commit_sha": "521691ae96ff54980fb5a402eb49b5b3ce9e2f37",
        "assignee": null,
        "assignees": [

        ],
        "milestone": null,
        "commits_url": "https://api.github.com/repos/google/go-github/pulls/472/commits",
        "review_comments_url": "https://api.github.com/repos/google/go-github/pulls/472/comments",
        "review_comment_url": "https://api.github.com/repos/google/go-github/pulls/comments{/number}",
        "comments_url": "https://api.github.com/repos/google/go-github/issues/472/comments",
        "statuses_url": "https://api.github.com/repos/google/go-github/statuses/d04745e0a3f0bf2eb9d5c01989a672b91bcd9791",
        "head": {
          "label": "nmiyake:updateLicense",
          "ref": "updateLicense",
          "sha": "d04745e0a3f0bf2eb9d5c01989a672b91bcd9791",
          "user": {
            "login": "nmiyake",
            "id": 4267425,
            "avatar_url": "https://avatars.githubusercontent.com/u/4267425?v=3",
            "gravatar_id": "",
            "url": "https://api.github.com/users/nmiyake",
            "html_url": "https://github.com/nmiyake",
            "followers_url": "https://api.github.com/users/nmiyake/followers",
            "following_url": "https://api.github.com/users/nmiyake/following{/other_user}",
            "gists_url": "https://api.github.com/users/nmiyake/gists{/gist_id}",
            "starred_url": "https://api.github.com/users/nmiyake/starred{/owner}{/repo}",
            "subscriptions_url": "https://api.github.com/users/nmiyake/subscriptions",
            "organizations_url": "https://api.github.com/users/nmiyake/orgs",
            "repos_url": "https://api.github.com/users/nmiyake/repos",
            "events_url": "https://api.github.com/users/nmiyake/events{/privacy}",
            "received_events_url": "https://api.github.com/users/nmiyake/received_events",
            "type": "User",
            "site_admin": false
          },
          "repo": {
            "id": 73579502,
            "name": "go-github",
            "full_name": "nmiyake/go-github",
            "owner": {
              "login": "nmiyake",
              "id": 4267425,
              "avatar_url": "https://avatars.githubusercontent.com/u/4267425?v=3",
              "gravatar_id": "",
              "url": "https://api.github.com/users/nmiyake",
              "html_url": "https://github.com/nmiyake",
              "followers_url": "https://api.github.com/users/nmiyake/followers",
              "following_url": "https://api.github.com/users/nmiyake/following{/other_user}",
              "gists_url": "https://api.github.com/users/nmiyake/gists{/gist_id}",
              "starred_url": "https://api.github.com/users/nmiyake/starred{/owner}{/repo}",
              "subscriptions_url": "https://api.github.com/users/nmiyake/subscriptions",
              "organizations_url": "https://api.github.com/users/nmiyake/orgs",
              "repos_url": "https://api.github.com/users/nmiyake/repos",
              "events_url": "https://api.github.com/users/nmiyake/events{/privacy}",
              "received_events_url": "https://api.github.com/users/nmiyake/received_events",
              "type": "User",
              "site_admin": false
            },
            "private": false,
            "html_url": "https://github.com/nmiyake/go-github",
            "description": "Go library for accessing the GitHub API",
            "fork": true,
            "url": "https://api.github.com/repos/nmiyake/go-github",
            "forks_url": "https://api.github.com/repos/nmiyake/go-github/forks",
            "keys_url": "https://api.github.com/repos/nmiyake/go-github/keys{/key_id}",
            "collaborators_url": "https://api.github.com/repos/nmiyake/go-github/collaborators{/collaborator}",
            "teams_url": "https://api.github.com/repos/nmiyake/go-github/teams",
            "hooks_url": "https://api.github.com/repos/nmiyake/go-github/hooks",
            "issue_events_url": "https://api.github.com/repos/nmiyake/go-github/issues/events{/number}",
            "events_url": "https://api.github.com/repos/nmiyake/go-github/events",
            "assignees_url": "https://api.github.com/repos/nmiyake/go-github/assignees{/user}",
            "branches_url": "https://api.github.com/repos/nmiyake/go-github/branches{/branch}",
            "tags_url": "https://api.github.com/repos/nmiyake/go-github/tags",
            "blobs_url": "https://api.github.com/repos/nmiyake/go-github/git/blobs{/sha}",
            "git_tags_url": "https://api.github.com/repos/nmiyake/go-github/git/tags{/sha}",
            "git_refs_url": "https://api.github.com/repos/nmiyake/go-github/git/refs{/sha}",
            "trees_url": "https://api.github.com/repos/nmiyake/go-github/git/trees{/sha}",
            "statuses_url": "https://api.github.com/repos/nmiyake/go-github/statuses/{sha}",
            "languages_url": "https://api.github.com/repos/nmiyake/go-github/languages",
            "stargazers_url": "https://api.github.com/repos/nmiyake/go-github/stargazers",
            "contributors_url": "https://api.github.com/repos/nmiyake/go-github/contributors",
            "subscribers_url": "https://api.github.com/repos/nmiyake/go-github/subscribers",
            "subscription_url": "https://api.github.com/repos/nmiyake/go-github/subscription",
            "commits_url": "https://api.github.com/repos/nmiyake/go-github/commits{/sha}",
            "git_commits_url": "https://api.github.com/repos/nmiyake/go-github/git/commits{/sha}",
            "comments_url": "https://api.github.com/repos/nmiyake/go-github/comments{/number}",
            "issue_comment_url": "https://api.github.com/repos/nmiyake/go-github/issues/comments{/number}",
            "contents_url": "https://api.github.com/repos/nmiyake/go-github/contents/{+path}",
            "compare_url": "https://api.github.com/repos/nmiyake/go-github/compare/{base}...{head}",
            "merges_url": "https://api.github.com/repos/nmiyake/go-github/merges",
            "archive_url": "https://api.github.com/repos/nmiyake/go-github/{archive_format}{/ref}",
            "downloads_url": "https://api.github.com/repos/nmiyake/go-github/downloads",
            "issues_url": "https://api.github.com/repos/nmiyake/go-github/issues{/number}",
            "pulls_url": "https://api.github.com/repos/nmiyake/go-github/pulls{/number}",
            "milestones_url": "https://api.github.com/repos/nmiyake/go-github/milestones{/number}",
            "notifications_url": "https://api.github.com/repos/nmiyake/go-github/notifications{?since,all,participating}",
            "labels_url": "https://api.github.com/repos/nmiyake/go-github/labels{/name}",
            "releases_url": "https://api.github.com/repos/nmiyake/go-github/releases{/id}",
            "deployments_url": "https://api.github.com/repos/nmiyake/go-github/deployments",
            "created_at": "2016-11-12T21:28:59Z",
            "updated_at": "2016-11-12T21:29:00Z",
            "pushed_at": "2016-11-12T21:33:10Z",
            "git_url": "git://github.com/nmiyake/go-github.git",
            "ssh_url": "git@github.com:nmiyake/go-github.git",
            "clone_url": "https://github.com/nmiyake/go-github.git",
            "svn_url": "https://github.com/nmiyake/go-github",
            "homepage": "http://godoc.org/github.com/google/go-github/github",
            "size": 1753,
            "stargazers_count": 0,
            "watchers_count": 0,
            "language": "Go",
            "has_issues": false,
            "has_downloads": true,
            "has_wiki": false,
            "has_pages": false,
            "forks_count": 0,
            "mirror_url": null,
            "open_issues_count": 0,
            "forks": 0,
            "open_issues": 0,
            "watchers": 0,
            "default_branch": "master"
          }
        },
        "base": {
          "label": "google:master",
          "ref": "master",
          "sha": "905f04725b7bcad81e475ce48474b87561ecc723",
          "user": {
            "login": "google",
            "id": 1342004,
            "avatar_url": "https://avatars.githubusercontent.com/u/1342004?v=3",
            "gravatar_id": "",
            "url": "https://api.github.com/users/google",
            "html_url": "https://github.com/google",
            "followers_url": "https://api.github.com/users/google/followers",
            "following_url": "https://api.github.com/users/google/following{/other_user}",
            "gists_url": "https://api.github.com/users/google/gists{/gist_id}",
            "starred_url": "https://api.github.com/users/google/starred{/owner}{/repo}",
            "subscriptions_url": "https://api.github.com/users/google/subscriptions",
            "organizations_url": "https://api.github.com/users/google/orgs",
            "repos_url": "https://api.github.com/users/google/repos",
            "events_url": "https://api.github.com/users/google/events{/privacy}",
            "received_events_url": "https://api.github.com/users/google/received_events",
            "type": "Organization",
            "site_admin": false
          },
          "repo": {
            "id": 10270722,
            "name": "go-github",
            "full_name": "google/go-github",
            "owner": {
              "login": "google",
              "id": 1342004,
              "avatar_url": "https://avatars.githubusercontent.com/u/1342004?v=3",
              "gravatar_id": "",
              "url": "https://api.github.com/users/google",
              "html_url": "https://github.com/google",
              "followers_url": "https://api.github.com/users/google/followers",
              "following_url": "https://api.github.com/users/google/following{/other_user}",
              "gists_url": "https://api.github.com/users/google/gists{/gist_id}",
              "starred_url": "https://api.github.com/users/google/starred{/owner}{/repo}",
              "subscriptions_url": "https://api.github.com/users/google/subscriptions",
              "organizations_url": "https://api.github.com/users/google/orgs",
              "repos_url": "https://api.github.com/users/google/repos",
              "events_url": "https://api.github.com/users/google/events{/privacy}",
              "received_events_url": "https://api.github.com/users/google/received_events",
              "type": "Organization",
              "site_admin": false
            },
            "private": false,
            "html_url": "https://github.com/google/go-github",
            "description": "Go library for accessing the GitHub API",
            "fork": false,
            "url": "https://api.github.com/repos/google/go-github",
            "forks_url": "https://api.github.com/repos/google/go-github/forks",
            "keys_url": "https://api.github.com/repos/google/go-github/keys{/key_id}",
            "collaborators_url": "https://api.github.com/repos/google/go-github/collaborators{/collaborator}",
            "teams_url": "https://api.github.com/repos/google/go-github/teams",
            "hooks_url": "https://api.github.com/repos/google/go-github/hooks",
            "issue_events_url": "https://api.github.com/repos/google/go-github/issues/events{/number}",
            "events_url": "https://api.github.com/repos/google/go-github/events",
            "assignees_url": "https://api.github.com/repos/google/go-github/assignees{/user}",
            "branches_url": "https://api.github.com/repos/google/go-github/branches{/branch}",
            "tags_url": "https://api.github.com/repos/google/go-github/tags",
            "blobs_url": "https://api.github.com/repos/google/go-github/git/blobs{/sha}",
            "git_tags_url": "https://api.github.com/repos/google/go-github/git/tags{/sha}",
            "git_refs_url": "https://api.github.com/repos/google/go-github/git/refs{/sha}",
            "trees_url": "https://api.github.com/repos/google/go-github/git/trees{/sha}",
            "statuses_url": "https://api.github.com/repos/google/go-github/statuses/{sha}",
            "languages_url": "https://api.github.com/repos/google/go-github/languages",
            "stargazers_url": "https://api.github.com/repos/google/go-github/stargazers",
            "contributors_url": "https://api.github.com/repos/google/go-github/contributors",
            "subscribers_url": "https://api.github.com/repos/google/go-github/subscribers",
            "subscription_url": "https://api.github.com/repos/google/go-github/subscription",
            "commits_url": "https://api.github.com/repos/google/go-github/commits{/sha}",
            "git_commits_url": "https://api.github.com/repos/google/go-github/git/commits{/sha}",
            "comments_url": "https://api.github.com/repos/google/go-github/comments{/number}",
            "issue_comment_url": "https://api.github.com/repos/google/go-github/issues/comments{/number}",
            "contents_url": "https://api.github.com/repos/google/go-github/contents/{+path}",
            "compare_url": "https://api.github.com/repos/google/go-github/compare/{base}...{head}",
            "merges_url": "https://api.github.com/repos/google/go-github/merges",
            "archive_url": "https://api.github.com/repos/google/go-github/{archive_format}{/ref}",
            "downloads_url": "https://api.github.com/repos/google/go-github/downloads",
            "issues_url": "https://api.github.com/repos/google/go-github/issues{/number}",
            "pulls_url": "https://api.github.com/repos/google/go-github/pulls{/number}",
            "milestones_url": "https://api.github.com/repos/google/go-github/milestones{/number}",
            "notifications_url": "https://api.github.com/repos/google/go-github/notifications{?since,all,participating}",
            "labels_url": "https://api.github.com/repos/google/go-github/labels{/name}",
            "releases_url": "https://api.github.com/repos/google/go-github/releases{/id}",
            "deployments_url": "https://api.github.com/repos/google/go-github/deployments",
            "created_at": "2013-05-24T16:42:58Z",
            "updated_at": "2016-11-12T17:11:49Z",
            "pushed_at": "2016-11-12T21:33:10Z",
            "git_url": "git://github.com/google/go-github.git",
            "ssh_url": "git@github.com:google/go-github.git",
            "clone_url": "https://github.com/google/go-github.git",
            "svn_url": "https://github.com/google/go-github",
            "homepage": "http://godoc.org/github.com/google/go-github/github",
            "size": 1753,
            "stargazers_count": 2038,
            "watchers_count": 2038,
            "language": "Go",
            "has_issues": true,
            "has_downloads": true,
            "has_wiki": false,
            "has_pages": false,
            "forks_count": 451,
            "mirror_url": null,
            "open_issues_count": 31,
            "forks": 451,
            "open_issues": 31,
            "watchers": 2038,
            "default_branch": "master"
          }
        },
        "_links": {
          "self": {
            "href": "https://api.github.com/repos/google/go-github/pulls/472"
          },
          "html": {
            "href": "https://github.com/google/go-github/pull/472"
          },
          "issue": {
            "href": "https://api.github.com/repos/google/go-github/issues/472"
          },
          "comments": {
            "href": "https://api.github.com/repos/google/go-github/issues/472/comments"
          },
          "review_comments": {
            "href": "https://api.github.com/repos/google/go-github/pulls/472/comments"
          },
          "review_comment": {
            "href": "https://api.github.com/repos/google/go-github/pulls/comments{/number}"
          },
          "commits": {
            "href": "https://api.github.com/repos/google/go-github/pulls/472/commits"
          },
          "statuses": {
            "href": "https://api.github.com/repos/google/go-github/statuses/d04745e0a3f0bf2eb9d5c01989a672b91bcd9791"
          }
        }
      }
    },
    "public": true,
    "created_at": "2016-11-12T21:35:46Z",
    "org": {
      "id": 1342004,
      "login": "google",
      "gravatar_id": "",
      "url": "https://api.github.com/orgs/google",
      "avatar_url": "https://avatars.githubusercontent.com/u/1342004?"
    }
  },
  {
    "id": "4857909641",
    "type": "CommitCommentEvent",
    "actor": {
      "id": 1924134,
      "login": "shurcooL",
      "display_login": "shurcooL",
      "gravatar_id": "",
      "url": "https://api.github.com/users/shurcooL",
      "avatar_url": "https://avatars.githubusercontent.com/u/1924134?"
    },
    "repo": {
      "id": 49927767,
      "name": "cweill/gotests",
      "url": "https://api.github.com/repos/cweill/gotests"
    },
    "payload": {
      "comment": {
        "url": "https://api.github.com/repos/cweill/gotests/comments/19798715",
        "html_url": "https://github.com/cweill/gotests/commit/a3c1a1257668d18c210c0a63299466e301f67c27#commitcomment-19798715",
        "id": 19798715,
        "user": {
          "login": "shurcooL",
          "id": 1924134,
          "avatar_url": "https://avatars.githubusercontent.com/u/1924134?v=3",
          "gravatar_id": "",
          "url": "https://api.github.com/users/shurcooL",
          "html_url": "https://github.com/shurcooL",
          "followers_url": "https://api.github.com/users/shurcooL/followers",
          "following_url": "https://api.github.com/users/shurcooL/following{/other_user}",
          "gists_url": "https://api.github.com/users/shurcooL/gists{/gist_id}",
          "starred_url": "https://api.github.com/users/shurcooL/starred{/owner}{/repo}",
          "subscriptions_url": "https://api.github.com/users/shurcooL/subscriptions",
          "organizations_url": "https://api.github.com/users/shurcooL/orgs",
          "repos_url": "https://api.github.com/users/shurcooL/repos",
          "events_url": "https://api.github.com/users/shurcooL/events{/privacy}",
          "received_events_url": "https://api.github.com/users/shurcooL/received_events",
          "type": "User",
          "site_admin": false
        },
        "position": null,
        "line": null,
        "path": null,
        "commit_id": "a3c1a1257668d18c210c0a63299466e301f67c27",
        "created_at": "2016-11-12T21:17:52Z",
        "updated_at": "2016-11-12T21:17:52Z",
        "body": "Why was that commit reverted?"
      }
    },
    "public": true,
    "created_at": "2016-11-12T21:17:52Z"
  },
  {
    "id": "4856428778",
    "type": "IssuesEvent",
    "actor": {
      "id": 1924134,
      "login": "shurcooL",
      "display_login": "shurcooL",
      "gravatar_id": "",
      "url": "https://api.github.com/users/shurcooL",
      "avatar_url": "https://avatars.githubusercontent.com/u/1924134?"
    },
    "repo": {
      "id": 55308508,
      "name": "dominikh/go-staticcheck",
      "url": "https://api.github.com/repos/dominikh/go-staticcheck"
    },
    "payload": {
      "action": "opened",
      "issue": {
        "url": "https://api.github.com/repos/dominikh/go-staticcheck/issues/16",
        "repository_url": "https://api.github.com/repos/dominikh/go-staticcheck",
        "labels_url": "https://api.github.com/repos/dominikh/go-staticcheck/issues/16/labels{/name}",
        "comments_url": "https://api.github.com/repos/dominikh/go-staticcheck/issues/16/comments",
        "events_url": "https://api.github.com/repos/dominikh/go-staticcheck/issues/16/events",
        "html_url": "https://github.com/dominikh/go-staticcheck/issues/16",
        "id": 188887388,
        "number": 16,
        "title": "\"defer rc.Close() or similar\" check doesn't detect bad calls of Close() on nested fields.",
        "user": {
          "login": "shurcooL",
          "id": 1924134,
          "avatar_url": "https://avatars.githubusercontent.com/u/1924134?v=3",
          "gravatar_id": "",
          "url": "https://api.github.com/users/shurcooL",
          "html_url": "https://github.com/shurcooL",
          "followers_url": "https://api.github.com/users/shurcooL/followers",
          "following_url": "https://api.github.com/users/shurcooL/following{/other_user}",
          "gists_url": "https://api.github.com/users/shurcooL/gists{/gist_id}",
          "starred_url": "https://api.github.com/users/shurcooL/starred{/owner}{/repo}",
          "subscriptions_url": "https://api.github.com/users/shurcooL/subscriptions",
          "organizations_url": "https://api.github.com/users/shurcooL/orgs",
          "repos_url": "https://api.github.com/users/shurcooL/repos",
          "events_url": "https://api.github.com/users/shurcooL/events{/privacy}",
          "received_events_url": "https://api.github.com/users/shurcooL/received_events",
          "type": "User",
          "site_admin": false
        },
        "labels": [

        ],
        "state": "open",
        "locked": false,
        "assignee": null,
        "assignees": [

        ],
        "milestone": null,
        "comments": 0,
        "created_at": "2016-11-12T02:35:48Z",
        "updated_at": "2016-11-12T02:35:48Z",
        "closed_at": null,
        "body": "One of the checks in README says:\r\n\r\n> - Don't ` + "`" + `defer rc.Close()` + "`" + ` before having checked the error returned by ` + "`" + `Open` + "`" + ` or similar.\r\n\r\nThe \"or similar\" statement implies it should work for detecting a similar error for ` + "`" + `http.Get` + "`" + ` and ` + "`" + `defer resp.Body.Close()` + "`" + `:\r\n\r\n` + "```" + `Go\r\npackage main\r\n\r\nimport (\r\n\t\"fmt\"\r\n\t\"io\"\r\n\t\"io/ioutil\"\r\n\t\"net/http\"\r\n)\r\n\r\nfunc main() {\r\n\tresp, err := http.Get(\"https://www.example.com/\")\r\n\tdefer resp.Body.Close()\r\n\tif err != nil {\r\n\t\tfmt.Println(err)\r\n\t\treturn\r\n\t}\r\n\t// read resp.Body or something\r\n\tio.Copy(ioutil.Discard, resp.Body)\r\n}\r\n` + "```" + `\r\n\r\nRunning ` + "`" + `staticcheck` + "`" + ` produces zero output:\r\n\r\n` + "```" + `\r\n$ staticcheck\r\n` + "```" + `\r\n\r\nI've already let @dominikh know about this, but reporting it here for posterity and tracking, and so that I can close an open window.\r\n\r\nRelated to golang/go#17780."
      }
    },
    "public": true,
    "created_at": "2016-11-12T02:35:48Z"
  },
  {
    "id": "4856269085",
    "type": "WatchEvent",
    "actor": {
      "id": 1924134,
      "login": "shurcooL",
      "display_login": "shurcooL",
      "gravatar_id": "",
      "url": "https://api.github.com/users/shurcooL",
      "avatar_url": "https://avatars.githubusercontent.com/u/1924134?"
    },
    "repo": {
      "id": 73430589,
      "name": "google/gops",
      "url": "https://api.github.com/repos/google/gops"
    },
    "payload": {
      "action": "started"
    },
    "public": true,
    "created_at": "2016-11-12T01:06:06Z",
    "org": {
      "id": 1342004,
      "login": "google",
      "gravatar_id": "",
      "url": "https://api.github.com/orgs/google",
      "avatar_url": "https://avatars.githubusercontent.com/u/1342004?"
    }
  },
  {
    "id": "4856254716",
    "type": "DeleteEvent",
    "actor": {
      "id": 1924134,
      "login": "shurcooL",
      "display_login": "shurcooL",
      "gravatar_id": "",
      "url": "https://api.github.com/users/shurcooL",
      "avatar_url": "https://avatars.githubusercontent.com/u/1924134?"
    },
    "repo": {
      "id": 73521088,
      "name": "shurcooL/spec",
      "url": "https://api.github.com/repos/shurcooL/spec"
    },
    "payload": {
      "ref": "patch-1",
      "ref_type": "branch",
      "pusher_type": "user"
    },
    "public": true,
    "created_at": "2016-11-12T00:59:56Z"
  }
]
`
