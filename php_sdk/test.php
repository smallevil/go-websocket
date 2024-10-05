<?php
include('./WsServer.php');

$wsServer = new WsServer('http://127.0.0.1', '16000');
//$ret = $wsServer->register('xxx', 'http://127.0.0.1');
//$ret = $wsServer->sendToClientId('test', 'KyiejPFDTE8EzxZ1vTXEj9No5IB8UQ/VbEzsWw/k52r+Xi7idDv+GwT8IgPN3kaO', 'smallevil', 0, 'msg', array('msg' => 'yyyy'));
//$ret = $wsServer->bindToGroup('test', 'group_test', 'M6hoOEzTuarqUC5ktoM5HKS94lCZeiqpBDJ3Pl4kHxhf3thYEdsMKCeLNBM7dQfO');
//$ret = $wsServer->setExtend('test', '400coYlg3Dbtk1WiRqUCxCeJAbBzv/gj58H9ryTni+dW0OP5j17SrY0EX3G3W1ms', 'cwsky');
//$ret = $wsServer->sendToGroup('test', '', 'send_user_id', 0, 'msg', 'data');
$ret = $wsServer->getClientInfo('test', 'd+wzXQkf2DcISjnGIfL4KWro69RdaLuk1w6nmRRay+a02YgK6L27/RNeO84X6ZLq');
print_r($ret);
?>