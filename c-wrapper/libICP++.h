/** @file */ 
#pragma once
#ifndef ICP

#include <libICP.h>
#include <string>
#include <vector>
#include <map>

// All strings are UTF-8
namespace ICP {
	class CAStore;
	class Error;
	class PFX;
	class Error;
	class CodedError;

	class Error {
	public:
		icp_err _err_ptr;
		std::string Message;
		Error(icp_err new_err_ptr);
		bool IsNull();
		~Error();
	};

	class CodedError: public Error {
	public:
		icp_errc _errc_ptr;
		std::string CodeStr;
		int Code;
		CodedError(icp_errc new_errc_ptr);
		~CodedError();	
	};

	class Cert {
	protected:
		void update();
	public:
		icp_cert _cert_ptr;
		std::string Subject, Issuer;
		std::string FingerPrintHuman, FingerPrintAlg;
		std::vector<uint8_t> FingerPrint;
		std::map<std::string, std::string> SubjectMap, IssuerMap;
		icp_time NotBefore, NotAfter;
		bool IsSelfSigned();
		bool IsCA();
		Cert(icp_cert new_cert_ptr);
		~Cert();	
	};

	int LoadCertsFromFile(std::string path, std::vector<Cert*> &certs, std::vector<CodedError*> &errs);
	int LoadCertsFromBytes(uint8_t *data, int n, std::vector<Cert*> &certs, std::vector<CodedError*> &errs);

	class CAStore {
	public:
		icp_store _store_ptr;
		std::string GetCachePath();
		void SetCachePath(std::string new_cache_path);
		bool GetAutoDownload();
		void SetAutoDownload(bool flag);
		bool GetDebug();
		void SetDebug(bool flag);
		int Verify(Cert *cert, std::vector<Cert*> &chain, std::vector<CodedError*> &errcs, std::vector<CodedError*> &warns);
		CodedError DownloadAll();
		Error AddAllCAsFromDir(std::string path);
		void AddAllCAsFromDirParallel(std::string path);
		std::vector<CodedError*> AddCA(Cert* cert);
		std::vector<CodedError*> AddTestingRootCA(Cert* cert);
		CAStore();
		CAStore(bool AutoDownload);
		~CAStore();	
	};

	class PFX: public Cert {
	public:
		icp_pfx _pfx_ptr;
		bool HasKey();
		CodedError SaveCertToFile(std::string path);
		CodedError SaveToFile(std::string path, std::string password);
		PFX(icp_pfx new_pfx_ptr);
		~PFX();	
	};

	PFX LoadPFXFromFile(std::string path, std::string password, CodedError &errc);
}
#endif //__LIBICP++__
