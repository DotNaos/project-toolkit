terraform {
  required_providers {
    github = {
      source  = "integrations/github"
      version = ">= 6.0.0"
    }
  }
}

variable "repository" {
  type = string
}

variable "pattern" {
  type    = string
  default = "main"
}

resource "github_branch_protection" "main" {
  repository_id = var.repository
  pattern       = var.pattern

  required_pull_request_reviews {
    required_approving_review_count = 1
  }
}
