<?php

declare(strict_types=1);

namespace Stock2Shop\Connector;

use Stock2Shop\Share\DTO;
use Stock2Shop\Share;

class Meta
{
    public const CHANNEL_META_URL_KEY = 'api_url';
    public const ALLOWED_CHANNEL_META = [
        self::CHANNEL_META_URL_KEY
    ];

    public array $map = [];

    public function __construct(DTO\Channel $channel)
    {
        foreach ($channel->meta as $m) {
            $this->map[$m->key] = $m->value;
        }
    }

    public function get(string $key): string|false
    {
        if (!in_array($key, self::ALLOWED_CHANNEL_META)) {
            return false;
        }
        if (isset($this->map[$key])) {
            return $this->map[$key];
        }
        return false;
    }
}
