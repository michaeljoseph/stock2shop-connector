<?php

declare(strict_types=1);

namespace Stock2Shop\Connector;

use Exception;
use GuzzleHttp\Exception\GuzzleException;
use Stock2Shop\Share\DTO;
use Stock2Shop\Share;

class Sync
{
    /**
     * @param DTO\ChannelProduct[] $channelProducts
     */
    public static function touchProducts(DemoAPI\API $api, array $channelProducts, DTO\Channel $channel): void
    {
        if (empty($channelProducts)) {
            return;
        }

        // transform
        try {
            $body = Transform::getDemoProducts($channelProducts);
        } catch (Exception) {
            SyncResults::setFailed($channelProducts);
            Logger::LogProductSyncFailed($channelProducts, 'Invalid Transform', $channel);
            return;
        }

        // Post to Demo API
        try {
            $dps = $api->postProducts($body);
        } catch (GuzzleException $e) {
            SyncResults::setFailed($channelProducts);
            Logger::LogProductSyncFailed($channelProducts, $e->getMessage(), $channel);
            return;
        }
        SyncResults::setSuccess($channelProducts, $dps);
    }

    /**
     * @param DTO\ChannelProduct[] $channelProducts
     */
    public static function deleteProducts(DemoAPI\API $api, array $channelProducts, DTO\Channel $channel): void
    {
        if (empty($channelProducts)) {
            return;
        }

        // transform
        try {
            $body = Transform::getDemoProductIDS($channelProducts);
        } catch (Exception) {
            SyncResults::setFailed($channelProducts);
            Logger::LogProductSyncFailed($channelProducts, 'Invalid Transform', $channel);
            return;
        }

        // Post to Demo API
        try {
            $api->deleteProducts($body);
        } catch (GuzzleException $e) {
            SyncResults::setFailed($channelProducts);
            Logger::LogProductSyncFailed($channelProducts, $e->getMessage(), $channel);
            return;
        }
        SyncResults::setDeleteSuccess($channelProducts);
    }
}
