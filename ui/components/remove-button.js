import { useState } from 'react'

import DeleteModal from './delete-modal'

export default function RemoveButton({
  children = 'Remove',
  onRemove,
  modalTitle,
  modalMessage,
}) {
  const [modalOpen, setModalOpen] = useState(false)

  return (
    <>
      <button
        data-testid='remove-button'
        type='button'
        onClick={() => setModalOpen(true)}
        className='inline-flex items-center rounded-md border border-transparent bg-black px-4 py-2 text-xs text-white shadow-sm hover:bg-gray-800'
      >
        {children}
      </button>
      <DeleteModal
        open={modalOpen}
        setOpen={setModalOpen}
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
