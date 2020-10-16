kubectl delete -f smfg-inventory-lb.yml
kubectl delete -f smfg-inventory-pod.yml
kubectl apply -f smfg-inventory-pod.yml
kubectl apply -f smfg-inventory-lb.yml
watch kubectl get all -n smfg-inventory -o wide
