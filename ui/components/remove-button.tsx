import { useState } from 'react'

import { RemoveButtonType } from '../lib/type'

import DeleteModal from './delete-modal'

interface IRemoveButtonProps {
  onRemove: () => void
  modalTitle: string
  modalMessage: string

  children?: string
  type?: RemoveButtonType.Button | RemoveButtonType.Link
}

export default function RemoveButton({
  onRemove,
  modalTitle,
  modalMessage,
  children = 'Remove',
  type = RemoveButtonType.Button,
}: IRemoveButtonProps) {

  const [modalOpen, setModalOpen] = useState<boolean>(false)

  return (
    <>
      {type === RemoveButtonType.Button && (
        <button
          data-testid='remove-button'
          type='button'
          onClick={() => setModalOpen(true)}
          className='inline-flex items-center rounded-md border border-gray-300 bg-white px-4 py-2 text-xs font-medium text-gray-700 shadow-sm hover:bg-gray-100'
        >
          {children}
        </button>
      )}
      {type === RemoveButtonType.Link && (
        <div className='group invisible rounded-md bg-transparent group-hover:visible'>
          <button
            onClick={() => setModalOpen(true)}
            className='flex items-center text-xs font-medium text-red-500 hover:text-red-500/50'
          >
            {children}
          </button>
        </div>
      )}

      <DeleteModal
        open={modalOpen}
        setOpen={() => setModalOpen(!modalOpen)}
        onSubmit={() => {
          onRemove()
          setModalOpen(false)
        }}
        title={modalTitle}
        message={modalMessage}
      />
    </>
  )
}
