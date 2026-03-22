"use client";

import { useState } from "react";
import dynamic from "next/dynamic";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Loader2 } from "lucide-react";
import { useStackMutations } from "@/lib/api/mutations";

const MonacoEditor = dynamic(() => import("@monaco-editor/react"), {
  ssr: false,
});

const DEFAULT_COMPOSE = `services:
  app:
    image: nginx:latest
    ports:
      - "8080:80"
    restart: unless-stopped
`;

interface CreateStackModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreated?: (stackId: string) => void;
}

export function CreateStackModal({
  open,
  onOpenChange,
  onCreated,
}: CreateStackModalProps) {
  const [name, setName] = useState("");
  const [compose, setCompose] = useState(DEFAULT_COMPOSE);
  const [env, setEnv] = useState("");
  const [nameError, setNameError] = useState("");

  const { createStack, isCreating } = useStackMutations();

  const validateName = (value: string) => {
    if (!/^[a-zA-Z0-9_-]+$/.test(value)) {
      setNameError(
        "Only letters, numbers, hyphens, and underscores are allowed",
      );
      return false;
    }
    setNameError("");
    return true;
  };

  const handleNameChange = (value: string) => {
    setName(value);
    if (value) validateName(value);
    else setNameError("");
  };

  const handleCreate = () => {
    if (!name.trim()) {
      setNameError("Name is required");
      return;
    }
    if (!validateName(name)) return;

    createStack.mutate(
      { name, compose, env },
      {
        onSuccess: (res) => {
          onOpenChange(false);
          setName("");
          setCompose(DEFAULT_COMPOSE);
          setEnv("");
          if (res.stack?.id) onCreated?.(res.stack.id);
        },
      },
    );
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="max-w-3xl max-h-[90vh] flex flex-col"
        data-testid="create-stack-modal"
      >
        <DialogHeader>
          <DialogTitle>Create Stack</DialogTitle>
          <DialogDescription>
            Create a new Docker Compose stack. The stack folder and files will
            be created in the configured stacks directory.
          </DialogDescription>
        </DialogHeader>

        <div className="flex flex-col gap-4 flex-1 min-h-0 overflow-y-auto">
          <div className="space-y-1.5">
            <Label htmlFor="stack-name">Stack Name</Label>
            <Input
              id="stack-name"
              placeholder="my-app"
              value={name}
              onChange={(e) => handleNameChange(e.target.value)}
              className={nameError ? "border-destructive" : ""}
              data-testid="stack-name-input"
            />
            {nameError && (
              <p className="text-xs text-destructive">{nameError}</p>
            )}
          </div>

          <Tabs defaultValue="compose" className="flex flex-col flex-1 min-h-0">
            <TabsList>
              <TabsTrigger value="compose">docker-compose.yml</TabsTrigger>
              <TabsTrigger value="env">.env (optional)</TabsTrigger>
            </TabsList>

            <TabsContent value="compose" className="flex-1 min-h-0">
              <div
                className="border border-border rounded-md overflow-hidden"
                style={{ height: 320 }}
              >
                <MonacoEditor
                  height="320px"
                  language="yaml"
                  value={compose}
                  onChange={(val) => setCompose(val ?? "")}
                  theme="vs-dark"
                  options={{
                    minimap: { enabled: false },
                    fontSize: 13,
                    lineNumbers: "on",
                    scrollBeyondLastLine: false,
                    automaticLayout: true,
                    tabSize: 2,
                  }}
                />
              </div>
            </TabsContent>

            <TabsContent value="env" className="flex-1 min-h-0">
              <div
                className="border border-border rounded-md overflow-hidden"
                style={{ height: 320 }}
              >
                <MonacoEditor
                  height="320px"
                  language="plaintext"
                  value={env}
                  onChange={(val) => setEnv(val ?? "")}
                  theme="vs-dark"
                  options={{
                    minimap: { enabled: false },
                    fontSize: 13,
                    lineNumbers: "on",
                    scrollBeyondLastLine: false,
                    automaticLayout: true,
                    tabSize: 2,
                  }}
                />
              </div>
            </TabsContent>
          </Tabs>
        </div>

        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={isCreating}
          >
            Cancel
          </Button>
          <Button
            onClick={handleCreate}
            disabled={isCreating || !name.trim()}
            data-testid="stack-create-submit"
          >
            {isCreating ? (
              <>
                <Loader2 className="size-4 mr-2 animate-spin" />
                Creating...
              </>
            ) : (
              "Create Stack"
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
