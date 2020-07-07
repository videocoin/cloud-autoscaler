```
helm upgrade -i --wait --set image.tag="v1.1.0-infra-99e802d" -n console autoscaler ./helm
```