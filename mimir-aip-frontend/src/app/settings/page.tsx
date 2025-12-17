"use client";

import { useState } from "react";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from "@/components/ui/card";
import APIKeysTab from "@/components/settings/APIKeysTab";
import PluginsTab from "@/components/settings/PluginsTab";

export default function SettingsPage() {
  const [activeTab, setActiveTab] = useState("api-keys");

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold text-orange">Settings</h1>
        <p className="text-gray-400 mt-2">
          Manage API keys, plugins, and system configuration
        </p>
      </div>

      {/* Tabs */}
      <Tabs value={activeTab} onValueChange={setActiveTab} className="w-full">
        <TabsList className="grid w-full max-w-md grid-cols-2 bg-blue">
          <TabsTrigger 
            value="api-keys"
            className="data-[state=active]:bg-orange data-[state=active]:text-navy"
          >
            API Keys
          </TabsTrigger>
          <TabsTrigger 
            value="plugins"
            className="data-[state=active]:bg-orange data-[state=active]:text-navy"
          >
            Plugins
          </TabsTrigger>
        </TabsList>

        <TabsContent value="api-keys" className="mt-6">
          <APIKeysTab />
        </TabsContent>

        <TabsContent value="plugins" className="mt-6">
          <PluginsTab />
        </TabsContent>
      </Tabs>
    </div>
  );
}
