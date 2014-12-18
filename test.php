<?php
date_default_timezone_set('Asia/Shanghai');

/** 获取当前时间戳，精确到毫秒 */

function microtime_float()
{
   list($usec, $sec) = explode(" ", microtime());
   return ((float)$usec + (float)$sec);
}


/** 格式化时间戳，精确到毫秒，x代表毫秒 */

function microtime_format($tag, $time)
{
   list($usec, $sec) = explode(".", $time);
   $date = date($tag,$usec);
   return str_replace('x', $sec, $date);
}

$fname = "test.com.log";
if(file_exists($fname))unlink($fname);



while(1) {
	file_put_contents($fname, microtime_format("Y-m-d H:i:s x", microtime_float()) . " test 中华人们共和国 log...\n", FILE_APPEND);
	usleep(100000);
}


