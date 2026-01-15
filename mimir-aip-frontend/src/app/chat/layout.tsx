import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "AI Agent Chat - Mimir AIP",
  description: "Chat with Mimir AI Assistant",
};

export default function ChatLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return children;
}
