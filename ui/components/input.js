export default function ({
  label,
  type,
  value,
  placeholder,
  handleInputChange,
  handleKeyDown,
  disabled = false,
  error,
  hasDropdownSelection = true,
  optionType,
  options,
  handleSelectOption,
  selectedItem
}) {
  return (
    <div>
      {label &&
        <label htmlFor='price' className='block text-sm font-medium text-white'>
          {label}
        </label>}
      <div className='relative rounded shadow-sm'>
        <input
          autoFocus
          type={type}
          value={value}
          className={`block w-full px-4 py-3 sm:text-sm border bg-transparent rounded-full focus:outline-none focus:ring focus:ring-cyan-600 disabled:opacity-30 ${error ? 'border-pink-500' : 'border-gray-800'}`}
          placeholder={placeholder}
          onChange={handleInputChange}
          onKeyDown={handleKeyDown}
          disabled={disabled}
        />
        {hasDropdownSelection &&
          <div className='absolute inset-y-0 right-6 flex items-center'>
            <label htmlFor={optionType} className='sr-only'>
              {optionType}
            </label>
            <select
              id={optionType}
              name={optionType}
              onChange={handleSelectOption}
              value={selectedItem}
              className='h-full py-0 pl-2 border-transparent bg-transparent text-white text-sm focus:outline-none'
            >
              {options.map((option) => (
                <option key={option} value={option}>{option}</option>
              ))}
            </select>
          </div>}
      </div>
    </div>
  )
}
