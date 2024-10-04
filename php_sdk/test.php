<?php
include('./WsServer.php');

$wsServer = new WsServer('http://127.0.0.1', '16000');
//$ret = $wsServer->register('xxx', 'http://127.0.0.1');
//$ret = $wsServer->sendToClientId('test', 'KyiejPFDTE8EzxZ1vTXEj9No5IB8UQ/VbEzsWw/k52r+Xi7idDv+GwT8IgPN3kaO', 'smallevil', 0, 'msg', array('msg' => 'yyyy'));
//$ret = $wsServer->bindToGroup('test', 'test_group', 'byGLYxM0xtLWE282VtDZuQbe+L0ZohmoGhxGVpfRn7WJPDBKw2PuMq6TdV37HF63');
//$ret = $wsServer->setExtend('test', '400coYlg3Dbtk1WiRqUCxCeJAbBzv/gj58H9ryTni+dW0OP5j17SrY0EX3G3W1ms', 'cwsky');
print_r($ret);
?>