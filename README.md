```text
This is a tools for k8s service forward.

How to use:

1.first, you need connect to k8s cluster.

  $ sudo k8s-forward connect
  +-----------+-------------------+------------------------------------+----------------+-------------------+
  | NAMESPACE | SERVICE_NAME      | POD_NAME                           | POD_IP         | POD_PORT          |
  +-----------+-------------------+------------------------------------+----------------+-------------------+
  | default   | portainer         | portainer-6c8cbf65b8-6j982         | 10.42.0.6      | 9000              |
  | default   | register          | register-85d5bd9596-zwx2l          | 10.42.0.7      | 5000              |
  | default   | mongoadmin        | mongoadmin-85f9d85749-kdsdf        | 10.42.2.57     | 80                |
  | default   | redisadmin        | redisadmin-5b558b7fdd-j444d        | 10.42.2.55     | 80                |
  | default   | redis-master      | redis-master-7bb968d545-4bzjn      | 10.42.9.2      | 6379              |
  | default   | redis-slave-1     | redis-slave-1-665c5b7f94-c8vmx     | 10.42.2.28     | 6379              |
  | default   | redis-slave-2     | redis-slave-2-75558d54bf-mjdms     | 10.42.3.36     | 6379              |
  | default   | redis-sentinel    | redis-sentinel-9b646cd4f-vcpb7     | 10.42.3.35     | 16379             |
  | default   | mongo-27017       | mongo-27017-67684bd7d7-trzl6       | 10.42.9.4      | 27017             |
  | default   | mongo-27018       | mongo-27018-9f7cdcd94-vxmtj        | 10.42.2.54     | 27018             |
  | default   | mongo-27019       | mongo-27019-654f76548f-88jc4       | 10.42.3.72     | 27019             |
  | default   | discover-server-0 | discover-server-0-b5d478f78-k6fzf  | 172.16.100.161 | 11000,12000,13000 |
  | default   | discover-server-1 | discover-server-1-58bbb7c9d5-tqgcz | 172.16.100.160 | 11000,12000,13000 |
  | default   | discover-server-2 | discover-server-2-5bdbc56657-vspl5 | 172.16.100.158 | 11000,12000,13000 |
  | default   | nginx             | web-0                              | 10.42.9.207    | 80                |
  +-----------+-------------------+------------------------------------+----------------+-------------------+


  now run the command:
  
  $ curl nginx
  
  that's all.

    
2.second, If you want to forward online traffic to the local port, 
  open another terminal and run the command:

  $ sudo k8s-forward switch deployment nginx -n default

  create a http server on localhost:
  
  $ python3 -m http.server --bind 0.0.0.0 80

  $ curl nginx

  if the nginx service's domain is nginx.example.com, you can use the command:
  
  $ curl nginx.example.com

  that's all.

3.recover the switch:

  $ sudo k8s-forward recover deployment nginx -n default

4.clean up:

  $ sudo k8s-forward clean
  
  it will run automatically when the program ends.
```
