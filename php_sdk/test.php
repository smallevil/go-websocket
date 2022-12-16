<?php
include('./WsServer.php');

$wsServer = new WsServer('http://127.0.0.1', '16000');
$ret = $wsServer->register('xxx', 'https://debug.tubie.net/test');
//$ret = $wsServer->register('xxx', 'http://127.0.0.1');
print_r($ret);
?>