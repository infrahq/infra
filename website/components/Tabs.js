import { useState, createContext } from 'react'

export const TabContext = createContext()

export default function Tabs ({ labels, children }) {
  const [currentTab, setCurrentTab] = useState(labels[0])

  return (
    <TabContext.Provider value={currentTab}>
      <div className='flex flex-wrap'>
        {labels.map(label => (
          <button
            key={label}
            className={`${label === currentTab ? 'text-white border-zinc-100' : 'text-zinc-500 border-transparent'} px-7 py-1.5 leading-loose border-b text-lg font-medium first-of-type:ml-0`}
            onClick={() => setCurrentTab(label)}
          >
            {label}
          </button>
        ))}
      </div>
      {children}
    </TabContext.Provider>
  )
}
