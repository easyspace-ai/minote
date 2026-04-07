import { useEffect, useRef } from "react";

import { useArtifacts } from "../artifacts";
import { useThread } from "../messages/context";

const ChatBox: React.FC<{ children: React.ReactNode; threadId: string }> = ({
  children,
  threadId,
}) => {
  const { thread } = useThread();
  const threadIdRef = useRef(threadId);

  const { setArtifacts, deselect } = useArtifacts();

  useEffect(() => {
    if (threadIdRef.current !== threadId) {
      threadIdRef.current = threadId;
      deselect();
    }
    setArtifacts(thread.values.artifacts ?? []);
  }, [threadId, deselect, setArtifacts, thread.values.artifacts]);

  return <div className="relative flex size-full min-h-0 flex-col">{children}</div>;
};

export { ChatBox };
