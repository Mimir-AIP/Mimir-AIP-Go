"use client";

import { useState } from "react";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import PluginsTab from "@/components/settings/PluginsTab";
import AIProvidersTab from "@/components/settings/AIProvidersTab";

export default function SettingsPage() {
  const [activeTab, setActiveTab] = useState("providers");

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold text-orange">Settings</h1>
        <p className="text-gray-400 mt-2">
          Manage AI providers, plugins, and system configuration
        </p>
      </div>

      {/* Tabs */}
      <Tabs value={activeTab} onValueChange={setActiveTab} className="w-full">
        <TabsList className="grid w-full max-w-md grid-cols-2 bg-blue">
          <TabsTrigger
            value="providers"
            className="data-[state=active]:bg-orange data-[state=active]:text-navy"
          >
            AI Providers
          </TabsTrigger>
          <TabsTrigger
            value="plugins"
            className="data-[state=active]:bg-orange data-[state=active]:text-navy"
          >
            Plugins
          </TabsTrigger>
        </TabsList>

        <TabsContent value="providers" className="mt-6">
          <AIProvidersTab />
        </TabsContent>

        <TabsContent value="plugins" className="mt-6">
          <PluginsTab />
        </TabsContent>
      </Tabs>
    </div>
  );
}
