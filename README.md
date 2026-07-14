# joss_smtp

Plugin oficial para habilitar el cliente de correos SMTP (`SmtpClient`) en el lenguaje Joss.

## Instalación

```bash
joss pub add joss_smtp
```

## Uso

```joss
use joss_smtp;

$client = new SmtpClient();
$client->send("correo@ejemplo.com", "Asunto", "Cuerpo del correo");
```
