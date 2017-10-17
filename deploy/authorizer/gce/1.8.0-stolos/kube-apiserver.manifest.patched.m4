changecom(`<unused>')

































{
"apiVersion": "v1",
"kind": "Pod",
"metadata": {
  "name":"kube-apiserver",
  "namespace": "kube-system",
  "annotations": {
    "scheduler.alpha.kubernetes.io/critical-pod": ""
  },
  "labels": {
    "tier": "control-plane",
    "component": "kube-apiserver"
  }
},
"spec":{
"hostNetwork": true,
"containers":[
    {
    "name": "kube-apiserver",
    "image": "APISERVER_IMAGE_NAME",
    "imagePullPolicy": "Never",
    "resources": {
      "requests": {
        "cpu": "250m"
      }
    },
    "command": [
                 "/bin/sh",
                 "-c",
                 "/usr/local/bin/kube-apiserver --v=2 --vmodule=webhook=9 --cloud-config=/etc/gce.conf --address=127.0.0.1 --allow-privileged=true --cloud-provider=gce --client-ca-file=/etc/srv/kubernetes/pki/ca-certificates.crt --etcd-servers=http://127.0.0.1:2379 --etcd-servers-overrides=/events#http://127.0.0.1:4002 --secure-port=443 --tls-cert-file=/etc/srv/kubernetes/pki/apiserver.crt --tls-private-key-file=/etc/srv/kubernetes/pki/apiserver.key --requestheader-client-ca-file=/etc/srv/kubernetes/pki/aggr_ca.crt --requestheader-allowed-names=aggregator --requestheader-extra-headers-prefix=X-Remote-Extra- --requestheader-group-headers=X-Remote-Group --requestheader-username-headers=X-Remote-User --proxy-client-cert-file=/etc/srv/kubernetes/pki/proxy_client.crt --proxy-client-key-file=/etc/srv/kubernetes/pki/proxy_client.key --enable-aggregator-routing=true --kubelet-client-certificate=/etc/srv/kubernetes/pki/apiserver-client.crt --kubelet-client-key=/etc/srv/kubernetes/pki/apiserver-client.key --service-account-key-file=/etc/srv/kubernetes/pki/serviceaccount.crt --token-auth-file=/etc/srv/kubernetes/known_tokens.csv --basic-auth-file=/etc/srv/kubernetes/basic_auth.csv --storage-backend=etcd3 --target-ram-mb=60 --service-cluster-ip-range=10.0.0.0/16 --etcd-quorum-read=false --admission-control=Initializers,NamespaceLifecycle,LimitRanger,ServiceAccount,PersistentVolumeLabel,DefaultStorageClass,DefaultTolerationSeconds,NodeRestriction,Priority,ResourceQuota --feature-gates=ExperimentalCriticalPodAnnotation=true --advertise-address=APISERVER_IP_ADDRESS --authorization-webhook-config-file=/etc/srv/kubernetes/authz.yaml --authorization-webhook-use-service-resolution --authorization-mode=Webhook,Node,RBAC --allow-privileged=true 1>>/var/log/kube-apiserver.log 2>&1"
               ],
    "env":[{"name": "KUBE_PATCH_CONVERSION_DETECTOR", "value": "false"}],
    "livenessProbe": {
      "httpGet": {
        "host": "127.0.0.1",
        "port": 8080,
        "path": "/healthz"
      },
      "initialDelaySeconds": 15,
      "timeoutSeconds": 15
    },
    "ports":[
      { "name": "https",
        "containerPort": 443,
        "hostPort": 443},{
       "name": "local",
        "containerPort": 8080,
        "hostPort": 8080}
        ],
    "volumeMounts": [
        {"name": "cloudconfigmount","mountPath": "/etc/gce.conf", "readOnly": true},
        
        
        
        
        
        
        
        { "name": "srvkube",
        "mountPath": "/etc/srv/kubernetes",
        "readOnly": true},
        { "name": "logfile",
        "mountPath": "/var/log/kube-apiserver.log",
        "readOnly": false},
        { "name": "auditlogfile",
        "mountPath": "/var/log/kube-apiserver-audit.log",
        "readOnly": false},
        { "name": "etcssl",
        "mountPath": "/etc/ssl",
        "readOnly": true},
        { "name": "usrsharecacerts",
        "mountPath": "/usr/share/ca-certificates",
        "readOnly": true},
        { "name": "varssl",
        "mountPath": "/var/ssl",
        "readOnly": true},
        { "name": "etcopenssl",
        "mountPath": "/etc/openssl",
        "readOnly": true},
        { "name": "etcpki",
        "mountPath": "/etc/srv/pki",
        "readOnly": true},
        { "name": "srvsshproxy",
        "mountPath": "/etc/srv/sshproxy",
        "readOnly": false}
      ]
    }
],
"volumes":[
  {"name": "cloudconfigmount","hostPath": {"path": "/etc/gce.conf", "type": "FileOrCreate"}},
  
  
  
  
  
  
  
  { "name": "srvkube",
    "hostPath": {
        "path": "/etc/srv/kubernetes"}
  },
  { "name": "logfile",
    "hostPath": {
        "path": "/var/log/kube-apiserver.log",
        "type": "FileOrCreate"}
  },
  { "name": "auditlogfile",
    "hostPath": {
        "path": "/var/log/kube-apiserver-audit.log",
        "type": "FileOrCreate"}
  },
  { "name": "etcssl",
    "hostPath": {
        "path": "/etc/ssl"}
  },
  { "name": "usrsharecacerts",
    "hostPath": {
        "path": "/usr/share/ca-certificates"}
  },
  { "name": "varssl",
    "hostPath": {
        "path": "/var/ssl"}
  },
  { "name": "etcopenssl",
    "hostPath": {
        "path": "/etc/openssl"}
  },
  { "name": "etcpki",
    "hostPath": {
        "path": "/etc/srv/pki"}
  },
  { "name": "srvsshproxy",
    "hostPath": {
        "path": "/etc/srv/sshproxy"}
  }
]
}}
