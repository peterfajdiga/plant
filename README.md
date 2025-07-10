# plant

<details>
<summary>$${\color{green}+}$$ resource "Interactive Terraform plan viewer" {</summary>
    $${\color{green}+}$$ description = "Displays your Terraform plan changes in a TUI tree view with collapsible branches"<br/>
   }
</details>

<details>
<summary>$${\color{orange}\sim}$$ resource "Terraform plan presentation" {</summary>
    $${\color{orange}\sim}$$ mode = "wall of text" $${\color{orange}\rightarrow}$$ "concise list of changes"<br/>
    $${\color{orange}\sim}$$ details_shown = "all" $${\color{orange}\rightarrow}$$ "user-selected"<br/>
   }
</details>

## Usage

```bash
$ plant terraform plan
```

```bash
$ terraform plan | plant
```

```bash
$ plant terraform apply
```

```bash
$ plant terraform destroy
```
