# Integtrating with ArgoCD to show probe status in GUI

Add the following section to `argocd-cm.yaml` to allow ArgoCD to understand the Networktest 
resource and show the probing status visually.

```
resource.customizations.useOpenLibs.edgeworks.no_Networktest: "true"
  resource.customizations: |
    edgeworks.no/Networktest:
      health.lua: |
        hs = {}
        hs.status = "Progressing"
        if obj.status ~= nil then
          if obj.status.lastResult == "Failed" then
            hs.status = "Degraded"
          else
            hs.status = "Healthy"
          end
          hs.message = obj.status.message
        end
        return hs
```
