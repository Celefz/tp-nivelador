# Informe de implementación

## Protocolo de comunicación

En el protocolo implementado, definimos tres tipos principales de mensajes:

* `TYPE_BETS`: referencia un conjunto de apuestas, todas juntas en un batch. 

    Las apuestas forman un paquete de la siguiente forma:

    ![Estructura de un bet](images/bet.png)

    Mientras los batch tiene esta estructura:

    ![Estructura de un batch](images/batch.png)

    Las apuestas, además de sus datos propios, llevan los campos `firstNameLen` y `lastNameLen` de manera que pueda saberse la longitud y parsearse estos campos sin la necesidad de usar un tamaño arbitrario y desperdiciar espacio.

    Por otro lado, cada batch lleva el identificador de agencia y un conjunto de apuestas serializadas. Además, llevan en su primer campo el tamaño del paquete entero. Esto permite leer la cantidad exacta de bytes que corresponde, ayudando al manejo de los problemas de short read y short write.

* `TYPE_ACK`: Permite al cliente saber que el servidor recibió su batch de apuestas enviado.

* `TYPE_END`: Sirve para que el servidor sepa cuando el cliente terminó de enviarle todas sus apuestas.

    ![Flujo de ACK y END](images/ack-end.png)

    Ambos tipos tienen la misma estructura, diferenciandose solo por el byte de tipo. En este caso, nuevamente están presentes los bytes de longitud, que facilitan el procesamiento de los paquetes.

En cuanto al flujo general del protocolo, el cliente envía un batch, el servidor lo procesa y confirma con ACK, y recién después el cliente continúa con el siguiente batch o con el mensaje final de cierre. Una vez recibido el mensaje final, el servidor procede a enviar las apuestas ganadoras.

![Diagrama del sistema](images/diagrama.jpg)


## Concurrencia

El servidor atiende cada conexión con un hilo independiente, lo que permite manejar varias agencias en paralelo sin bloquear el resto del sistema. El almacenamiento compartido se protege con un lock para evitar condiciones de carrera al registrar apuestas y al consultar el estado del lote.

Además, se usa una condición compartida para esperar el quorum de agencias necesarias antes de continuar con el cálculo de resultados. Así, el servidor no avanza antes de tener la cantidad mínima de participantes requerida y, al mismo tiempo, puede seguir respondiendo a nuevas conexiones mientras espera.

## Graceful shutdown

El cliente usa un context y un goroutine para detectar SIGTERM y cerrar la conexión de forma controlada. De esa manera, el cierre no se trata como un error fatal sino como una interrupción prevista del flujo de trabajo.

En el servidor, el evento de apagado marca el cierre global del servicio, notifica las condiciones en espera, cierra sockets activos y hace join de los threads. Esto evita que el proceso quede bloqueado en tareas pendientes o en espera de clientes que ya no deberían seguir activos.






