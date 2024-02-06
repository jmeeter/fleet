import React, { useContext } from "react";

import hostAPI from "services/entities/hosts";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import Checkbox from "components/forms/fields/Checkbox";
import { NotificationContext } from "context/notification";
import { AxiosError } from "axios";

const baseClass = "lock-modal";

interface ILockModalProps {
  id: number;
  platform: string;
  onClose: () => void;
}

const LockModal = ({ id, platform, onClose }: ILockModalProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const [lockChecked, setLockChecked] = React.useState(false);
  const [isLocking, setIsLocking] = React.useState(false);

  const onLock = async () => {
    setIsLocking(true);
    try {
      await hostAPI.lockHost(id);
      renderFlash("success", "Host locked successfully!");
    } catch (error) {
      const err = error as AxiosError;
      renderFlash("error", err.message);
    }
    onClose();
    setIsLocking(false);
  };

  return (
    <Modal className={baseClass} title="Lock host" onExit={onClose}>
      <>
        <div className={`${baseClass}__modal-content`}>
          <p>Lock a host when it needs to be returned to your organization.</p>
          {platform === "darwin" && (
            <p>Fleet will generate a six-digit unlock PIN.</p>
          )}
          <div className={`${baseClass}__confirm-message`}>
            <span>
              <b>Please check to confirm:</b>
            </span>
            <Checkbox
              wrapperClassName={`${baseClass}__lock-checkbox`}
              value={lockChecked}
              onChange={(value: boolean) => setLockChecked(value)}
            >
              I wish to lock <b>Test Host</b>
            </Checkbox>
          </div>
        </div>

        <div className="modal-cta-wrap">
          <Button
            type="button"
            onClick={onLock}
            variant="brand"
            className="delete-loading"
            disabled={!lockChecked}
            isLoading={isLocking}
          >
            Done
          </Button>
          <Button onClick={onClose} variant="inverse">
            Cancel
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default LockModal;
