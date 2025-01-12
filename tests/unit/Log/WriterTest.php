<?php

declare(strict_types=1);

namespace Stock2Shop\Tests\Connector\unit\Log;

use PHPUnit\Framework\TestCase;
use Stock2Shop\Connector\Config\Environment;
use Stock2Shop\Connector\Config\LoaderArray;
use Stock2Shop\Connector\Log\Writer;
use Stock2Shop\Share\DTO;

class WriterTest extends TestCase
{
    public function testWrite()
    {
        // set environment
        Environment::set(
            new LoaderArray([
                'LOG_CHANNEL'      => 'Share',
                'LOG_FS_DIR'       => sprintf('%s/../../output/', __DIR__),
                'LOG_FS_FILE_NAME' => 'system.log'
            ])
        );

        // test writing logs to file
        // clean test file
        $file = Environment::getLogFSDIR() . Environment::getLogFSFileName();
        if (file_exists($file)) {
            unlink($file);
        }

        $writer = new Writer();
        $log    = new DTO\Log([
            'message'   => 'Hello World',
            'client_id' => 21,
            'log_to_es' => true,
            'level'     => DTO\Log::LOG_LEVEL_INFO,
            'origin'    => 'Connector X',
            'context'   => [
                'foo' => 'bar',
                'baz' => 'bat'
            ]
        ]);

        $writer->write($log);
        $log->level = DTO\Log::LOG_LEVEL_WARNING;
        $writer->write($log);
        $log->level = DTO\Log::LOG_LEVEL_ERROR;
        $writer->write($log);
        $logs  = file_get_contents(Environment::getLogFSDIR() . Environment::getLogFSFileName());
        $parts = explode("\n", $logs);
        // 4 lines, one is space at end
        $this->assertCount(4, $parts);
        $this->assertEquals('', $parts[3]);
        $obj = json_decode($parts[0], true);
        $this->assertEquals('info', $obj['level']);
        $this->assertEquals(21, $obj['client_id']);
        $this->assertEquals('bar', $obj['context']['foo']);
        $this->assertArrayHasKey('datetime', $obj);
    }
}
