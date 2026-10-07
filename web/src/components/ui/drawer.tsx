import { Dialog, DialogContent } from "./dialog";

export const Drawer = Dialog;
export function DrawerContent(
  props: React.ComponentProps<typeof DialogContent>,
) {
  return <DialogContent {...props} placement="right" />;
}
