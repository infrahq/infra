import { Combobox } from '@headlessui/react'
import { useRef } from 'react'

import Badge from './badge'
import ComboboxItem from './combobox-item'

export default function TypeaheadCombobox({
  selectedEmails,
  setSelectedEmails,
  onRemove,
  inputRef,
  setQuery,
  filteredEmail,
  onKeyDownEvent,
}) {
  const button = useRef()

  return (
    <Combobox
      as='div'
      className='relative flex-1'
      onChange={e => {
        setSelectedEmails([...selectedEmails, e])
      }}
    >
      <div className='flex flex-auto flex-wrap'>
        {selectedEmails?.map(i => (
          <Badge key={i.id} onRemove={() => onRemove(i)}>
            {i.name}
          </Badge>
        ))}
        <div className='flex-1'>
          <Combobox.Input
            type='search'
            ref={inputRef}
            className='relative border-none bg-transparent text-xs text-gray-300 placeholder:italic focus:border-transparent focus:ring-0'
            onChange={e => setQuery(e.target.value)}
            onFocus={() => {
              button.current?.click()
            }}
            onKeyDown={e => onKeyDownEvent(e.key)}
            placeholder={selectedEmails.length === 0 ? 'Add users' : ''}
          />
        </div>
      </div>
      {filteredEmail.length > 0 && (
        <Combobox.Options className='= absolute z-10 mt-4 max-h-60 w-56 overflow-auto rounded-md border border-gray-100 bg-white py-1 text-xs shadow-xl shadow-gray-300/20 focus:outline-none'>
          {filteredEmail?.map(f => (
            <Combobox.Option
              key={f.id}
              value={f}
              className={({ active }) =>
                `relative cursor-default select-none py-2 px-3 text-gray-600 ${
                  active ? 'bg-gray-700' : ''
                }`
              }
            >
              <ComboboxItem
                title={f.name}
                subtitle={f.user ? 'User' : f.group ? 'Group' : ''}
              />
            </Combobox.Option>
          ))}
        </Combobox.Options>
      )}
      <Combobox.Button className='hidden' ref={button} />
    </Combobox>
  )
}
