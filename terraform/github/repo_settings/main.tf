terraform {
  required_providers {
    github = {
      source  = "integrations/github"
      version = ">= 6.0.0"
    }
  }
}

variable "name" {
  type = string
}

resource "github_repository" "this" {
  name                   = var.name
  visibility             = "private"
  has_issues             = true
  delete_branch_on_merge = true
}
