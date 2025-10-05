terraform {
  required_providers {
    null = {
      source = "hashicorp/null"
      version = "3.1.0"
    }
  }
}

resource "null_resource" "cluster" {
  triggers = {
    cluster_name = "my-cluster"
  }
}

resource "null_resource" "app" {
  triggers = {
    # This creates an implicit dependency on the 'cluster' resource
    cluster_id = null_resource.cluster.id
  }
}