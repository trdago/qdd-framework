### Configuración SQS local

Instalar Docker y ejecutar en la consola:

1. Descarga imagen

```
docker pull softwaremill/elasticmq
```
2. Ejecuta SQS localmente 
```
docker run -p 9324:9324 -p 9325:9325 softwaremill/elasticmq
```

Puedes revisar el estado de las cola en: http://localhost:9325/

Más información en: https://hub.docker.com/r/softwaremill/elasticmq